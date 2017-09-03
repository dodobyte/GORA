package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/veandco/go-sdl2/img"
	mix "github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
	ttf "github.com/veandco/go-sdl2/ttf"
)

const (
	wWindow  = 1366
	hWindow  = 768
	wFire    = 27
	hFire    = 9
	wRocket  = 21
	hRocket  = 57
	wEnmFire = 10
	hEnmFire = 10
	wExplos  = 60
	hExplos  = 50
)

var shp *ship

var renderer *sdl.Renderer

var starstext *sdl.Texture
var rocketex *sdl.Texture
var firetex *sdl.Texture
var enmfiretex *sdl.Texture
var explostext *sdl.Texture

var expsound *mix.Chunk

var score int32

var iLevel int32
var levels []*level

var stars [50]star

type star struct {
	x, y   int32
	w, h   int32
	sprite int32
}

type level struct {
	maxPoint   int32
	enmPrice   int32
	enmPeriod  uint32
	firePeriod uint32
	fireRange  float64
	fireRGB    [3]uint8
	enemyRGB   [3]uint8
}

type bullet struct {
	v, t int32
	x, y int32
}

type ship struct {
	x, y    int32
	w, h    int32
	vx, vy  int32
	life    int32
	rocket  int32
	rockets []*bullet
	bullets []*bullet
	texture *sdl.Texture
}

type enemy struct {
	w, h    int32
	tnew    uint32
	insects []*insect
	bullets []*bullet
	texture *sdl.Texture
}

type insect struct {
	x, y   int32
	killed bool
	tfire  uint32
	tdeath int32
}

type boss struct {
	x, y    int32
	w, h    int32
	life    int32
	tfire   uint32
	killed  bool
	bullets []*bullet
	texture *sdl.Texture
}

func (b *boss) fire() {
	if sdl.GetTicks()-b.tfire < 1000 {
		return
	}
	b.tfire = sdl.GetTicks()
	barrel := [4]int32{8, 52, 110, 153}
	for _, x := range barrel {
		bu := &bullet{5, 0, b.x + x, b.y + b.h}
		b.bullets = append(b.bullets, bu)
	}
}

func (b *boss) move() {

	rboss := &sdl.Rect{b.x, b.y, b.w, b.h}
	rship := &sdl.Rect{shp.x, shp.y, shp.w, shp.h}

	for _, bu := range shp.bullets {
		rb := &sdl.Rect{bu.x, bu.y, wEnmFire, hEnmFire}
		if rb.HasIntersection(rboss) {
			renderExplosion(bu.x, bu.y)
			bu.x = -100
			bu.y = -100
			b.life--
			if b.life <= 0 {
				b.killed = true
			}
			score += 100
		}
	}
	for _, r := range shp.rockets {
		rb := &sdl.Rect{r.x, r.y, wRocket, hRocket}
		if rb.HasIntersection(rboss) {
			renderExplosion(r.x, r.y)
			r.x = -100
			r.y = -100
			b.life -= 5
			if b.life <= 0 {
				b.killed = true
			}
			score += 500
		}
	}

	switch {
	case b.x < shp.x:
		b.x += 2
	case b.x > shp.x:
		b.x -= 2
	}
	switch {
	case b.x < 0:
		b.x = 0
	case b.x > wWindow-b.w:
		b.x = wWindow - b.w
	}

	active := b.bullets[:0]
	for _, bu := range b.bullets {
		rb := &sdl.Rect{bu.x, bu.y, wEnmFire, hEnmFire}
		if rb.HasIntersection(rship) {
			renderExplosion(bu.x, bu.y)
			shp.life--
			bu.x = -100
			bu.y = -100
		}
		if bu.v*bu.t < hWindow {
			bu.t++
			bu.y += bu.v
			active = append(active, bu)
		}
	}
	b.bullets = active
}

func (b *boss) render() {
	renderer.Copy(b.texture, nil, &sdl.Rect{b.x, b.y, b.w, b.h})

	for _, bu := range b.bullets {
		rect := &sdl.Rect{bu.x, bu.y, wEnmFire, hEnmFire}
		renderer.Copy(enmfiretex, nil, rect)
	}
	if b.killed {
		renderExplosion(b.x, b.y)
		renderExplosion(b.x+b.w, b.y)
		renderExplosion(b.x, b.y+b.h)
		renderExplosion(b.x+b.w, b.y+b.h)
	}
}

func (s *ship) fire() {
	b := &bullet{10, 0, s.x + (s.w-wFire)/2, s.y}
	s.bullets = append(s.bullets, b)
}

func (s *ship) fireRocket() {
	x := s.x
	if rand.Intn(2) == 1 {
		x += s.w - wRocket
	}
	b := &bullet{1, 0, x, s.y}
	s.rockets = append(s.rockets, b)
}

func (s *ship) move() {
	x := s.x + s.vx
	y := s.y + s.vy
	switch {
	case x < 0:
		s.x = 0
	case x > wWindow-s.w:
		s.x = wWindow - s.w
	case y < hWindow/2:
		s.y = hWindow / 2
	case y > hWindow-s.h:
		s.y = hWindow - s.h
	default:
		s.x = x
		s.y = y
	}
	active := s.bullets[:0]
	for _, b := range s.bullets {
		if b.v*b.t < hWindow {
			b.t++
			b.y -= b.v
			active = append(active, b)
		}
	}
	s.bullets = active

	active = s.rockets[:0]
	for _, r := range s.rockets {
		if r.v*r.t*r.t/6 < hWindow {
			r.t++
			r.y -= r.v * r.t / 3
			active = append(active, r)
		}
	}
	s.rockets = active
}

func (s *ship) render() {
	src := &sdl.Rect{rand.Int31n(4) * s.w, 0, s.w, s.h}
	renderer.Copy(s.texture, src, &sdl.Rect{s.x, s.y, s.w, s.h})

	for _, b := range s.bullets {
		renderer.Copy(firetex, nil, &sdl.Rect{b.x, b.y, wFire, hFire})
	}
	for _, r := range s.rockets {
		src := &sdl.Rect{rand.Int31n(2) * wRocket, 0, wRocket, hRocket}
		renderer.Copy(rocketex, src, &sdl.Rect{r.x, r.y, wRocket, hRocket})
	}
	if s.life == 0 {
		renderExplosion(s.x, s.y)
	}
}

func (e *enemy) create() {
	if len(e.insects) >= 20 {
		return
	}
	period := levels[iLevel].enmPeriod + uint32(len(e.insects)*200)
	if sdl.GetTicks()-e.tnew < period {
		return
	}
	e.tnew = sdl.GetTicks()
	x := rand.Int31n(wWindow - e.w)
	y := rand.Int31n(hWindow/2 - e.h)
	e.insects = append(e.insects, &insect{x: x, y: y})
}

func (e *enemy) move() {
	rb := &sdl.Rect{0, 0, wFire, hFire}
	ri := &sdl.Rect{0, 0, e.w, e.h}
	for _, i := range e.insects {
		ri.X = i.x
		ri.Y = i.y
		for _, b := range shp.bullets {
			rb.X = b.x
			rb.Y = b.y
			if rb.HasIntersection(ri) {
				b.x = -100
				b.y = -100
				i.tdeath = 1
				score += levels[iLevel].enmPrice
				break
			}
		}
		i.x += (rand.Int31n(5) - 2)
		switch {
		case i.x < 0:
			i.x = 0
		case i.x > wWindow-e.w:
			i.x = wWindow - e.w
		}
	}

	rs := &sdl.Rect{shp.x, shp.y, shp.w, shp.h}
	active := e.bullets[:0]

	for _, b := range e.bullets {
		rb := &sdl.Rect{b.x, b.y, wEnmFire, hEnmFire}
		if rb.HasIntersection(rs) {
			renderExplosion(b.x, b.y)
			shp.life--
			b.x = -100
			b.y = -100
		}
		if b.v*b.t < hWindow {
			b.t++
			b.y += b.v
			active = append(active, b)
		}
	}
	e.bullets = active
}

func (e *enemy) fire() {
	lvl := levels[iLevel]
	for _, i := range e.insects {
		if math.Abs(float64(shp.x-i.x)) < lvl.fireRange {
			if sdl.GetTicks()-i.tfire < lvl.firePeriod {
				continue
			}
			i.tfire = sdl.GetTicks()
			b := &bullet{5, 0, i.x + (e.w-wEnmFire)/2, i.y + e.h}
			e.bullets = append(e.bullets, b)
		}
	}
}

func (e *enemy) render() {
	ergb := levels[iLevel].enemyRGB
	frgb := levels[iLevel].fireRGB

	for _, i := range e.insects {
		e.texture.SetColorMod(ergb[0], ergb[1], ergb[2])
		renderer.Copy(e.texture, nil, &sdl.Rect{i.x, i.y, e.w, e.h})
		if i.tdeath > 0 {
			i.tdeath++
			if i.tdeath == 8 {
				i.killed = true
			}
			renderExplosion(i.x, i.y)
		}
	}
	alive := e.insects[:0]
	for _, i := range e.insects {
		if !i.killed {
			alive = append(alive, i)
		}
	}
	e.insects = alive
	for _, b := range e.bullets {
		rb := &sdl.Rect{b.x, b.y, wEnmFire, hEnmFire}
		enmfiretex.SetColorMod(frgb[0], frgb[1], frgb[2])
		renderer.Copy(enmfiretex, nil, rb)
	}
}

func renderStars() {
	for i := range stars {
		s := &stars[i]
		s.y += 3
		if s.y > hWindow {
			s.x = rand.Int31n(wWindow)
			s.y = rand.Int31n(3)
			s.sprite = rand.Int31n(4)
		}
		src := &sdl.Rect{s.w * s.sprite, 0, s.w, s.h}
		dst := &sdl.Rect{s.x, s.y, s.w, s.h}
		renderer.Copy(starstext, src, dst)
	}
}

func renderExplosion(x, y int32) {
	expsound.Play(-1, 0)
	src := &sdl.Rect{rand.Int31n(3) * wExplos, 0, wExplos, hExplos}
	renderer.Copy(explostext, src, &sdl.Rect{x, y, wExplos, hExplos})
}

func renderScore() {
	point := fmt.Sprintf("Score     %d", score)
	lives := fmt.Sprintf("Lives     %d", shp.life)
	rocket := fmt.Sprintf("Rockets   %d", shp.rocket)

	WriteText(point, 20, "red", 10, 10)
	WriteText(lives, 20, "green", 10, 40)
	WriteText(rocket, 20, "blue", 10, 70)
}

func TextSize(text string, size int) (int32, int32) {
	font, err := ttf.OpenFont("res/Go-Mono.ttf", size)
	if err != nil {
		panic(err)
	}
	defer font.Close()
	color := sdl.Color{255, 255, 255, 255}
	stext, err := font.RenderUTF8_Blended(text, color)
	if err != nil {
		panic(err)
	}
	defer stext.Free()
	return stext.W, stext.H
}

func WriteText(text string, size int, color string, x, y int32) {
	var clr sdl.Color
	switch color {
	case "white":
		clr = sdl.Color{255, 255, 255, 255}
	case "red":
		clr = sdl.Color{255, 100, 100, 255}
	case "green":
		clr = sdl.Color{100, 255, 100, 255}
	case "blue":
		clr = sdl.Color{100, 100, 255, 255}
	default:
		return
	}
	font, err := ttf.OpenFont("res/Go-Mono.ttf", size)
	if err != nil {
		panic(err)
	}
	defer font.Close()
	stext, err := font.RenderUTF8_Blended(text, clr)
	if err != nil {
		panic(err)
	}
	defer stext.Free()
	texture, err := renderer.CreateTextureFromSurface(stext)
	if err != nil {
		panic(err)
	}
	defer texture.Destroy()
	renderer.Copy(texture, nil, &sdl.Rect{x, y, stext.W, stext.H})
}

func createLevels() {
	level1 := &level{
		maxPoint:   100,
		enmPrice:   10,
		enmPeriod:  2000,
		fireRange:  100,
		firePeriod: 1000,
		fireRGB:    [3]uint8{150, 255, 255},
		enemyRGB:   [3]uint8{100, 255, 0},
	}
	level2 := &level{
		maxPoint:   1300,
		enmPrice:   60,
		enmPeriod:  1500,
		fireRange:  150,
		firePeriod: 750,
		fireRGB:    [3]uint8{255, 255, 255},
		enemyRGB:   [3]uint8{255, 255, 255},
	}
	level3 := &level{
		maxPoint:   7300,
		enmPrice:   200,
		enmPeriod:  1000,
		fireRange:  200,
		firePeriod: 500,
		fireRGB:    [3]uint8{255, 150, 150},
		enemyRGB:   [3]uint8{255, 50, 255},
	}
	levels = append(levels, level1)
	levels = append(levels, level2)
	levels = append(levels, level3)
}

func levelUp() bool {
	if score >= levels[iLevel].maxPoint {
		iLevel++
		if len(levels) == int(iLevel) {
			iLevel--
			return false
		}
	}
	return true
}

func menu(pause bool) {
	menuid := 1
	selected := 0
	showabout := false

	renderMenu := func() {
		w := int32(300)
		h := int32(300)
		x := (wWindow - w) / 2
		y := (hWindow - h) / 2
		renderer.SetDrawColor(255, 255, 0, 255)
		renderer.DrawRect(&sdl.Rect{x, y, w, h})
		renderer.DrawRect(&sdl.Rect{x + 2, y + 2, w - 4, h - 4})
		renderer.SetDrawColor(0, 0, 0, 255)

		if showabout {
			name := "G.O.R.A."
			company := "dodobyte d.o.o."
			credits := "Credits"
			site1 := "golang.org"
			site2 := "libsdl.org"
			site3 := "github.com/veandco/go-sdl2"

			wt, ht := TextSize(name, 32)
			y += ht + 10
			WriteText(name, 32, "white", x+(w-wt)/2, y)

			wt, ht = TextSize(company, 20)
			y += ht + 30
			WriteText(company, 20, "green", x+(w-wt)/2, y)

			wt, ht = TextSize(credits, 20)
			y += ht + 20
			WriteText(credits, 20, "blue", x+(w-wt)/2, y)

			wt, ht = TextSize(site1, 14)
			y += ht + 20
			WriteText(site1, 14, "red", x+(w-wt)/2, y)

			wt, ht = TextSize(site2, 14)
			y += ht + 10
			WriteText(site2, 14, "red", x+(w-wt)/2, y)

			wt, ht = TextSize(site3, 14)
			y += ht + 10
			WriteText(site3, 14, "red", x+(w-wt)/2, y)
			return
		}

		play := "PLAY"
		if pause {
			play = "RESUME"
		}
		about := "ABOUT"
		exit := "EXIT"

		switch menuid {
		case 1:
			play = "= " + play + " ="
		case 2:
			about = "= " + about + " ="
		case 3:
			exit = "= " + exit + " ="
		}

		wt, ht := TextSize(play, 32)
		y += ht + 25
		WriteText(play, 32, "green", x+(w-wt)/2, y)

		wt, ht = TextSize(about, 32)
		y += ht + 25
		WriteText(about, 32, "blue", x+(w-wt)/2, y)

		wt, ht = TextSize(exit, 32)
		y += ht + 25
		WriteText(exit, 32, "red", x+(w-wt)/2, y)
	}

	for quit := false; !quit; {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				quit = true
			case *sdl.KeyDownEvent:
				if t.Repeat == 0 {
					switch key := t.Keysym.Sym; key {
					case sdl.K_UP:
						menuid--
						if menuid < 1 {
							menuid = 1
						}
					case sdl.K_DOWN:
						menuid++
						if menuid > 3 {
							menuid = 3
						}
					case sdl.K_RETURN:
						selected = menuid
					case sdl.K_ESCAPE:
						switch {
						case showabout:
							showabout = false
							selected = 0
						case pause:
							quit = true
						default:
							os.Exit(0)
						}
					}
				}
			}
		}
		switch selected {
		case 1:
			quit = true
		case 2:
			showabout = true
		case 3:
			os.Exit(0)
		}
		renderer.Clear()
		renderStars()
		renderMenu()
		renderer.Present()
	}
}

func loadMusic(name string) *mix.Music {
	music, err := mix.LoadMUS(name)
	if err != nil {
		panic(err)
	}
	return music
}

func loadWAV(name string) *mix.Chunk {
	chunk, err := mix.LoadWAV(name)
	if err != nil {
		panic(err)
	}
	return chunk
}

func createWindow(w, h int, name string) *sdl.Window {
	undef := sdl.WINDOWPOS_UNDEFINED
	var wndtype uint32 = sdl.WINDOW_FULLSCREEN
	sdl.ShowCursor(sdl.DISABLE)
	window, err := sdl.CreateWindow(name, undef, undef, w, h, wndtype)
	if err != nil {
		panic(err)
	}
	return window
}

func createRenderer(win *sdl.Window) *sdl.Renderer {
	var flag uint32 = sdl.RENDERER_ACCELERATED | sdl.RENDERER_PRESENTVSYNC
	renderer, err := sdl.CreateRenderer(win, -1, flag)
	if err != nil {
		panic(err)
	}
	return renderer
}

func loadTexture(file string, renderer *sdl.Renderer) *sdl.Texture {
	image, err := img.Load(file)
	if err != nil {
		panic(err)
	}
	image.SetColorKey(1, sdl.MapRGB(image.Format, 0xff, 0x00, 0xff))

	texture, err := renderer.CreateTextureFromSurface(image)
	if err != nil {
		panic(err)
	}
	image.Free()
	return texture
}

func init() {
	runtime.LockOSThread()
}

func main() {

	rand.Seed(time.Now().UnixNano())

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	if flag := img.Init(img.INIT_PNG); flag != img.INIT_PNG {
		panic("img.Init failed")
	}
	defer img.Quit()

	if err := ttf.Init(); err != nil {
		panic(err)
	}
	defer ttf.Quit()

	if err := mix.Init(mix.INIT_MP3); err != nil {
		panic(err)
	}
	defer mix.Quit()

	if err := mix.OpenAudio(44100, mix.DEFAULT_FORMAT, 2, 2048); err != nil {
		panic(err)
	}
	defer mix.CloseAudio()
	mix.VolumeMusic(10)

	theme := loadMusic("res/hucum.mp3")
	defer theme.Free()

	expsound = loadWAV("res/explosion.wav")
	defer expsound.Free()
	expsound.Volume(16)

	shot := loadWAV("res/shot.wav")
	defer shot.Free()
	shot.Volume(4)

	if err := theme.Play(-1); err != nil {
		panic(err)
	}

	window := createWindow(wWindow, hWindow, "dodobyte G.O.R.A. 1.0")
	defer window.Destroy()

	renderer = createRenderer(window)
	defer renderer.Destroy()

	giftex := loadTexture("res/star.png", renderer)
	defer giftex.Destroy()

	starstext = loadTexture("res/stars.png", renderer)
	defer starstext.Destroy()

	rocketex = loadTexture("res/rocket.png", renderer)
	defer rocketex.Destroy()

	firetex = loadTexture("res/fire.png", renderer)
	defer firetex.Destroy()

	enmfiretex = loadTexture("res/enmfire.png", renderer)
	defer enmfiretex.Destroy()

	explostext = loadTexture("res/explosion.png", renderer)
	defer explostext.Destroy()

	tship := loadTexture("res/ship.png", renderer)
	defer tship.Destroy()

	tenemy := loadTexture("res/enemy.png", renderer)
	defer tenemy.Destroy()

	tboss := loadTexture("res/boss.png", renderer)
	defer tboss.Destroy()

	var bos *boss
	var enm *enemy

	createFighters := func() {
		shp = &ship{x: 350, y: 450, w: 63, h: 90, texture: tship, life: 1}
		shp.rockets = make([]*bullet, 0, 4096)
		shp.bullets = make([]*bullet, 0, 4096)

		bos = &boss{x: 350, y: 50, w: 175, h: 94, life: 100, texture: tboss}
		bos.bullets = make([]*bullet, 0, 4096)

		enm = &enemy{texture: tenemy, w: 61, h: 62}
		enm.insects = make([]*insect, 0, 4096)
		enm.bullets = make([]*bullet, 0, 4096)
	}
	createFighters()

	createLevels()

	for i := range stars {
		stars[i].x = rand.Int31n(wWindow)
		stars[i].y = rand.Int31n(hWindow)
		stars[i].w = 10
		stars[i].h = 5
		stars[i].sprite = rand.Int31n(4)
	}

	type gift struct {
		x, y    int32
		w, h    int32
		inside  string
		counter int32
		present bool
	}
	gft := gift{}
	setGift := func(inside string) {
		gft.w = int32(39)
		gft.h = int32(39)
		gft.x = rand.Int31n(wWindow - gft.w)
		gft.y = rand.Int31n(hWindow/2-gft.h) + hWindow/2
		gft.inside = inside
		gft.present = true
		gft.counter++
	}
	renderGift := func() {
		rect := &sdl.Rect{gft.x, gft.y, gft.w, gft.h}
		renderer.Copy(giftex, nil, rect)
	}

	var showmenu, pause, bossplays bool

	reset := func() {
		createFighters()
		score = 0
		showmenu = true
		pause = false
		bossplays = false
		iLevel = 0
		gft.present = false
		gft.counter = 0
	}
	reset()

	for {
		if showmenu {
			menu(pause)
			showmenu = false
		}
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.KeyDownEvent:
				if t.Repeat == 0 {
					switch key := t.Keysym.Sym; key {
					case sdl.K_LEFT:
						shp.vx = -8
					case sdl.K_RIGHT:
						shp.vx = +8
					case sdl.K_UP:
						shp.vy = -10
					case sdl.K_DOWN:
						shp.vy = +10
					case sdl.K_ESCAPE:
						pause = true
						showmenu = true
					case sdl.K_SPACE:
						shp.fire()
						shot.Play(-1, 0)
					case sdl.K_LCTRL, sdl.K_RCTRL:
						if shp.rocket > 0 {
							shp.fireRocket()
							shot.Play(-1, 0)
							shp.rocket--
						}
					}
				}
			case *sdl.KeyUpEvent:
				if t.Repeat == 0 {
					switch key := t.Keysym.Sym; key {
					case sdl.K_LEFT, sdl.K_RIGHT:
						shp.vx = 0
					case sdl.K_UP, sdl.K_DOWN:
						shp.vy = 0
					}
				}
			}
		}
		if iLevel == 1 && gft.counter == 0 {
			setGift("life")
		}
		renderer.Clear()

		renderStars()

		shp.move()
		shp.render()
		if shp.life == 0 {
			w, h := TextSize("GAME OVER", 64)
			WriteText("GAME OVER", 64, "red", (wWindow-w)/2, (hWindow-h)/2)
			renderer.Present()
			reset()
			sdl.Delay(2000)
			continue
		}

		if bossplays {
			bos.move()
			bos.fire()
			bos.render()
			if bos.killed {
				w, h := TextSize("YOU WIN!", 64)
				x, y := (wWindow-w)/2, (hWindow-h)/2
				WriteText("YOU WIN!", 64, "green", x, y)
				renderer.Present()
				reset()
				sdl.Delay(3000)
			}
		} else {
			enm.create()
		}
		enm.move()
		enm.fire()
		enm.render()

		if gft.present {
			rship := &sdl.Rect{shp.x, shp.y, shp.w, shp.h}
			rgift := &sdl.Rect{gft.x, gft.y, gft.w, gft.h}
			if rship.HasIntersection(rgift) {
				switch gft.inside {
				case "life":
					shp.life++
				case "rocket":
					shp.rocket += 10
				}
				gft.present = false
			}
			renderGift()
		}

		renderScore()
		renderer.Present()

		if !levelUp() {
			bossplays = true
			if gft.counter == 1 {
				setGift("rocket")
			}
		}
	}
}
