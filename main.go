package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jakecoffman/cp"
)

const (
	screenWidth  = 350
	screenHeight = 700
	iterations = 10
	gravity = 75
	playerFriction = 1.5
	playerElasticity = 0
	playerRadius = 0.0
	objectsFriction = 1.5
	objectsMass = 100.0
	objectsRadius = 1.0
	objectsElasticity = 0.0
	objectsMaxSize = 75
	objectsMinSize = 25
	zeroScore = screenHeight - 25
	numBody = 13
)

// For drawing the rigid bodies on screen with Ebiten
type ebitenDrawer struct {
	screen *ebiten.Image
	upperPoint float64
	lost bool
	col color.Color
}

func (d *ebitenDrawer) DrawCircle(pos cp.Vector, angle, radius float64, outline, fill cp.FColor, data interface{}) {}

func (d *ebitenDrawer) DrawSegment(a, b cp.Vector, fill cp.FColor, data interface{}) {
}

func (d *ebitenDrawer) DrawFatSegment(a, b cp.Vector, radius float64, outline, fill cp.FColor, data interface{}) {
}

func (d *ebitenDrawer) DrawPolygon(count int, verts []cp.Vector, radius float64, outline, fill cp.FColor, data interface{}) {
	col := d.col
	if d.lost {
		col = color.RGBA{150, 150, 150, 255}
	}
	lastX := verts[len(verts)-1].X
	lastY := verts[len(verts)-1].Y
	for _, v := range verts {
		ebitenutil.DrawLine(d.screen, lastX, lastY, v.X, v.Y, col)
		lastX = v.X
		lastY = v.Y
		if v.Y < d.upperPoint && v.X >= 0 && v.X <= screenWidth {
			d.upperPoint = v.Y
		}
	}
}

func (d *ebitenDrawer) DrawDot(size float64, pos cp.Vector, fill cp.FColor, data interface{}) {}
func (d *ebitenDrawer) Flags() uint {return 0}
func (d *ebitenDrawer) OutlineColor() cp.FColor {return cp.FColor{R: 0, G: 0, B: 0, A: 0}}
func (d *ebitenDrawer) ShapeColor(shape *cp.Shape, data interface{}) cp.FColor {return cp.FColor{R: 0, G: 0, B: 0, A: 0}}
func (d *ebitenDrawer) ConstraintColor() cp.FColor {return cp.FColor{R: 0, G: 0, B: 0, A: 0}}
func (d *ebitenDrawer) CollisionPointColor() cp.FColor {return cp.FColor{R: 0, G: 0, B: 0, A: 0}}
func (d *ebitenDrawer) Data() interface{} {return 0}

// Function that builds one box
func makeBox(x, y float64) (*cp.Shape, int) {
	w := float64(rand.Intn(objectsMaxSize - objectsMinSize + 1) + objectsMinSize)
	h := rand.Intn(objectsMaxSize - objectsMinSize + 1) + objectsMinSize
	mass := objectsMass * w * float64(h)
	body := cp.NewBody(mass, cp.MomentForBox(mass, w, float64(h)))
	body.SetPosition(cp.Vector{X: x, Y: y})

	shape := cp.NewBox(body, w, float64(h), objectsRadius)
	shape.SetElasticity(objectsElasticity)
	shape.SetFriction(objectsFriction)

	return shape, h
}

// The game structure
type Game struct {
	start bool
	space *cp.Space
	playerBody *cp.Body
	mouseX int
	readyMouseX bool
	upperPoint float64
	wasComputed bool
	score uint
	bestscore uint
	isStable bool
	lastUpperPoint float64
	upperSince int
	isLost bool
	maxscore bool
	oncemaxscore bool
}

// Method for reseting the game
func (g *Game) Reset() {

		space := cp.NewSpace()
		space.Iterations = iterations
		space.SetGravity(cp.Vector{X: 0, Y: gravity})

		var shape *cp.Shape

		playerBody := cp.NewKinematicBody()
		playerBody.SetPosition(cp.Vector{X: float64(screenWidth/2), Y:float64(screenHeight-10)})
		space.AddBody(playerBody)

		playerShape := cp.NewBox(
			playerBody,
			float64(screenWidth/2),
			10,
			playerRadius,
		)
		playerShape.SetElasticity(playerElasticity)
		playerShape.SetFriction(playerFriction)
		space.AddShape(playerShape)

		borderBodyLeft := cp.NewStaticBody()
		borderBodyLeft.SetPosition(cp.Vector{X:0, Y:0})
		space.AddBody(borderBodyLeft)
		shape = cp.NewSegment(borderBodyLeft, cp.Vector{X:0, Y:0}, cp.Vector{X:0, Y:float64(screenHeight)}, 1)
		shape.SetElasticity(objectsElasticity)
		shape.SetFriction(objectsFriction)
		space.AddShape(shape)

		borderBodyRight := cp.NewStaticBody()
		borderBodyRight.SetPosition(cp.Vector{X:float64(screenWidth), Y:0})
		space.AddBody(borderBodyRight)
		shape = cp.NewSegment(borderBodyRight, cp.Vector{X:0, Y:0}, cp.Vector{X:0, Y:float64(screenHeight)}, 1)
		shape.SetElasticity(objectsElasticity)
		shape.SetFriction(objectsFriction)
		space.AddShape(shape)

		addedBody := 0
		currentHeight := screenHeight - 25
		for addedBody < numBody {
				currentHeight -= 15
				shape, height := makeBox(float64(screenWidth/2), float64(currentHeight))
				currentHeight -= height
				space.AddBody(shape.Body())
				space.AddShape(shape)
				addedBody++
		}

		g.space = space
		g.playerBody = playerBody
		g.upperPoint = screenHeight
		g.score = 0
		g.readyMouseX = false
		g.wasComputed = false
		g.isStable = false
		g.lastUpperPoint = screenHeight
		g.upperSince = 0
		g.isLost = false
		g.maxscore = false

}

// Creation of a new game
func NewGame() *Game {

	g := &Game{}
	g.Reset()
	g.start = true
	return g

}

// Update method for Ebiten game interface
func (g *Game) Update() error {


	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		return errors.New("The end")
	}

	if g.start {

		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.start = false
		}
	} else {

if !g.isLost && !g.isStable {
	x, _ := ebiten.CursorPosition()
	if g.readyMouseX {
		velocity := 60*(x - g.mouseX)
		g.playerBody.SetVelocity(float64(velocity), 0)
	} else if x != 0 {
		g.readyMouseX = true
	}
	g.mouseX = x

	if g.wasComputed {
	addToScore := zeroScore - int(g.upperPoint)
	if addToScore > 0 {
		oldScore := g.score
		g.score += uint(addToScore)
		if g.score < oldScore {
			g.maxscore = true
			g.oncemaxscore = true
		}
		if g.score > g.bestscore {
			g.bestscore = g.score
		}
	} else {
		g.isLost = g.score > 0
		log.Print(zeroScore, addToScore, g.upperPoint)
	}

	if g.upperPoint - g.lastUpperPoint >= 5 || g.lastUpperPoint - g.upperPoint >= 5 {
		g.lastUpperPoint = g.upperPoint
		g.upperSince = 0
	} else {
		g.upperSince++
	}

	if g.upperSince >= 180 {
		g.isStable = true
	}
}

	g.upperPoint = screenHeight
	g.wasComputed = false


		g.space.Step(1.0 / 60)
} else {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.Reset()
	}
}


}
	return nil
}

// Draw method for Ebiten game interface
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 25, 255})

	if g.start {
		ebitenutil.DebugPrintAt(screen, "Stay Unstable!", screenWidth/2-40, 70)
		ebitenutil.DebugPrintAt(screen, "Developped in 12h for Ludum Dare 49.", 70, 100)

		ebitenutil.DebugPrintAt(screen, "Keep the tower unstable as long as you can by using your", 10, 150)
		ebitenutil.DebugPrintAt(screen, "moose to move the base of it.", 10, 160)

		ebitenutil.DebugPrintAt(screen, "The higher the tower and the longer it remains unstable,", 10, 180)
		ebitenutil.DebugPrintAt(screen, "the more points you get.", 10, 190)

		ebitenutil.DebugPrintAt(screen, "What score will you be able to reach?", 10, 210)

		ebitenutil.DebugPrintAt(screen, "At any time, press escape to quit.", 10, screenHeight-70)


				ebitenutil.DebugPrintAt(screen, "Press enter to start.", 220, screenHeight-20)
			return
	}

	// Draw the score level
	ebitenutil.DrawLine(screen, float64(screenWidth - 15), 0, float64(screenWidth - 15), zeroScore, color.RGBA{0, 0, 255, 255})
	for y := 0; y <= zeroScore; y++ {
		if (zeroScore - y) % 50 == 0 {
			ebitenutil.DrawLine(screen, float64(screenWidth-10), float64(y), float64(screenWidth-20), float64(y), color.RGBA{0, 0, 255, 255})
		}
		if (zeroScore - y) % 100 == 0 && (zeroScore - y) != 0{
			ebitenutil.DebugPrintAt(screen, fmt.Sprint((zeroScore - y)/100 * 100), screenWidth - 25, y - 8)
		}
	}

  // Draw the boxes
	drawer := &ebitenDrawer{
		screen: screen,
		upperPoint: g.upperPoint,
		lost: g.isStable || g.isLost,
	}

	g.space.EachBody(func(body *cp.Body) {
		body.EachShape(func(shape *cp.Shape) {
			if body == g.playerBody {
				drawer.col = color.RGBA{255, 0, 0, 255}
			} else {
				drawer.col = color.RGBA{255, 255, 255, 255}
			}
			cp.DrawShape(shape, drawer)
		})
	})

	g.wasComputed = true

	// Draw the upper point
	g.upperPoint = drawer.upperPoint
	if g.upperPoint > zeroScore + 5 {
		g.upperPoint = zeroScore + 5
	}
	ebitenutil.DrawLine(screen, 0, g.upperPoint, screenWidth, g.upperPoint, color.RGBA{255, 0, 0, 255})


	// Draw the score
	if g.oncemaxscore {
	ebitenutil.DebugPrintAt(screen, fmt.Sprint("Record MAXSCORE"), 10, 10)
	} else {
	ebitenutil.DebugPrintAt(screen, fmt.Sprint("Record ", g.bestscore), 10, 10)
}
if g.maxscore{
	ebitenutil.DebugPrintAt(screen, fmt.Sprint("Score  MAXSCORE"), 10, 20)
} else {
	ebitenutil.DebugPrintAt(screen, fmt.Sprint("Score  ", g.score), 10, 20)
}

	// Inform about stability
	if g.isStable {
		ebitenutil.DebugPrintAt(screen, "Stable.", screenWidth/2-20, screenHeight/2)
		ebitenutil.DebugPrintAt(screen, "You lose.", screenWidth/2-25, screenHeight/2+10)
		ebitenutil.DebugPrintAt(screen, "Press enter to restart.", 110, screenHeight/2+40)
	} else if g.isLost {
		ebitenutil.DebugPrintAt(screen, "No more objects.", screenWidth/2-45, screenHeight/2)
		ebitenutil.DebugPrintAt(screen, "You lose.", screenWidth/2-25, screenHeight/2+10)
		ebitenutil.DebugPrintAt(screen, "Press enter to restart.", 110, screenHeight/2+40)
	}

	//ebitenutil.DebugPrintAt(screen, fmt.Sprintf("TPS: %0.2f", ebiten.CurrentTPS()), 200, 0)
}

// Layout method for Ebiten game interface
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// Run the game with Ebiten
func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Stay unstable!")
	ebiten.SetCursorMode(ebiten.CursorModeCaptured)
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
