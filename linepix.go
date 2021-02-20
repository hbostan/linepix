package linepix

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"

	"github.com/nfnt/resize"
)

type ColorChannel int

const (
	RedChannel ColorChannel = iota
	GreenChannel
	BlueChannel
	AllChannels
)

type Line struct {
	Points  []image.Point
	Weight  int
	Channel ColorChannel
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func clamp(value, lower, upper int) uint8 {
	return uint8(min(max(value, lower), upper))
}

func ReadImage(imageName string) image.Image {
	imgHandle, err := os.Open(imageName)
	if err != nil {
		log.Println("⚠ Cannot open input image.", err)
		return nil
	}
	img, ext, err := image.Decode(imgHandle)
	if err != nil {
		log.Println("⚠ Couldn't decode the image. Make sure it's a JPEG or PNG image. 🖼", err)
		return nil
	}
	log.Printf("👨🏻‍💻 Image Decoded (%v)\n", ext)
	return img
}

func SaveImage(fileName string, img image.Image) error {
	handle, err := os.Create(fileName)
	if err != nil {
		log.Println("⚠ Can't create file to save image.", err)
		return err
	}
	png.Encode(handle, img)
	return nil
}

func ResizeImage(img image.Image, width, height uint) image.Image {
	return resize.Resize(width, height, img, resize.Lanczos3)
}

func MakeGrayscale(img image.Image) *image.Gray {
	grayscale := image.NewGray(img.Bounds())
	bounds := grayscale.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayscale.Set(x, y, img.At(x, y))
		}
	}
	return grayscale
}

func MakeRGBA(img image.Image) *image.RGBA {
	rgba := image.NewRGBA(img.Bounds())
	bounds := rgba.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
			if rgba.Pix[rgba.PixOffset(x, y)+3] < 0xff {
				rgba.Pix[rgba.PixOffset(x, y)+0] = 0xff
				rgba.Pix[rgba.PixOffset(x, y)+1] = 0xff
				rgba.Pix[rgba.PixOffset(x, y)+2] = 0xff
				rgba.Pix[rgba.PixOffset(x, y)+3] = 0xff
			}
		}
	}
	return rgba
}

func SplitImage(img *image.RGBA) (*image.Gray, *image.Gray, *image.Gray) {
	bounds := img.Bounds()
	red := image.NewGray(bounds)
	green := image.NewGray(bounds)
	blue := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			red.Pix[red.PixOffset(x, y)] = img.Pix[img.PixOffset(x, y)+0]
			green.Pix[green.PixOffset(x, y)] = img.Pix[img.PixOffset(x, y)+1]
			blue.Pix[blue.PixOffset(x, y)] = img.Pix[img.PixOffset(x, y)+2]
		}
	}
	return red, green, blue
}

func plotLine(start, end image.Point) []image.Point {
	var points []image.Point
	var dx, sx, dy, sy int
	current := start
	if dx, sx = end.X-start.X, 1; dx < 0 {
		dx = -dx
		sx = -sx
	}
	if dy, sy = end.Y-start.Y, -1; dy >= 0 {
		dy = -dy
		sy = -sy
	}
	err := dx + dy
	for {
		points = append(points, image.Point{current.X, current.Y})
		if current == end {
			return points
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			current.X += sx
		}
		if e2 <= dx {
			err += dx
			current.Y += sy
		}
	}
}

func findEdgeIntersections(bounds image.Rectangle, p image.Point, slope float64) (image.Point, image.Point) {
	width := bounds.Dx()
	height := bounds.Dy()

	start, end := image.Point{-1, -1}, image.Point{-1, -1}
	t := int(float64(p.X) + (1.0/slope)*float64(p.Y))
	r := int(float64(p.Y) - slope*float64(width-p.X))
	b := int(float64(p.X) - (1/slope)*float64(height-p.Y))
	l := int(float64(p.Y) + slope*float64(p.X))
	if t >= 0 && t < width {
		// Line starts from the top edge of the image
		start = image.Point{t, 0}
		// Line ends on the left edge of the image
		if l >= 0 && l < height {
			end = image.Point{0, l}
		}
		// Line ends on the bottom edge of the image
		if b >= 0 && b < width {
			end = image.Point{b, bounds.Max.Y - 1}
		}
		// Line ends on the right edge of the image
		if r >= 0 && r < height {
			end = image.Point{bounds.Max.X - 1, r}
		}
	} else if l >= 0 && l < height {
		// Line starts from the left edge of the image
		start = image.Point{0, l}
		// Line ends on the bottom edge of the image
		if b >= 0 && b < width {
			end = image.Point{b, bounds.Max.Y - 1}
		}
		// Line ends on the right edge of the image
		if r >= 0 && r < height {
			end = image.Point{bounds.Max.X - 1, r}
		}
	} else if b >= 0 && b < width {
		// Line starts from the bottom edge of the image
		start = image.Point{b, bounds.Max.Y - 1}
		// Line ends on the right edge of the image
		if r >= 0 && r < height {
			end = image.Point{bounds.Max.X - 1, r}
		}
	} else {
		log.Fatalf("💀 ERROR: Line doesn't go through the image!\n"+
			"\tPoint: (%v,%v) Slope: %v\n"+
			"\tImageSize: %vx%v\n"+
			"\tIntersections: T:%v R:%v B:%v L:%v\n",
			p.X, p.Y, slope, bounds.Dx(), bounds.Dy(), t, r, b, l)
	}
	return start, end
}

func DrawLine(img image.Image, line Line, color bool) {
	if color {
		rgbaImg := img.(*image.RGBA)
		for _, point := range line.Points {
			pixOffset := rgbaImg.PixOffset(point.X, point.Y)
			r := int(rgbaImg.Pix[pixOffset+0])
			g := int(rgbaImg.Pix[pixOffset+1])
			b := int(rgbaImg.Pix[pixOffset+2])
			switch line.Channel {
			case 0: //Red
				rgbaImg.Pix[pixOffset+0] = clamp(r-line.Weight, 0, 0xff)
			case 1: //Green
				rgbaImg.Pix[pixOffset+1] = clamp(g-line.Weight, 0, 0xff)
			case 2: //Blue
				rgbaImg.Pix[pixOffset+2] = clamp(b-line.Weight, 0, 0xff)
			default:
				rgbaImg.Pix[pixOffset+0] = clamp(r-line.Weight, 0, 0xff)
				rgbaImg.Pix[pixOffset+1] = clamp(g-line.Weight, 0, 0xff)
				rgbaImg.Pix[pixOffset+2] = clamp(b-line.Weight, 0, 0xff)
			}
			rgbaImg.Pix[pixOffset+3] = 0xff
		}
	} else {
		grayImg := img.(*image.Gray)
		for _, point := range line.Points {
			pixOffset := grayImg.PixOffset(point.X, point.Y)
			gray := int(grayImg.Pix[pixOffset])
			grayImg.Pix[pixOffset] = clamp(gray-line.Weight, 0, 0xff)
		}
	}
}

func findDarkestPixels(img *image.Gray) []image.Point {
	var points []image.Point
	bounds := img.Bounds()
	points = append(points, image.Point{0, 0})
	lum := img.Pix[0]
	for y := 1; y < bounds.Max.Y; y++ {
		for x := 1; x < bounds.Max.X; x++ {
			cur := img.Pix[y*img.Stride+x]
			if cur < lum {
				points = points[:0]
				points = append(points, image.Point{x, y})
				lum = cur
			} else if cur == lum {
				points = append(points, image.Point{x, y})
			}
		}
	}
	return points
}

func lineLuminosity(img *image.Gray, line Line) int {
	var lum int
	for _, point := range line.Points {
		lum += int(img.Pix[point.Y*img.Stride+point.X])
	}
	if lum < 0 {
		log.Panicln("⚠ Line luminosity overflow!")
	}
	return lum
}

func findDarkestLine(img *image.Gray, p image.Point, numTests int) Line {
	var line Line
	m := (rand.Float64() - 0.5) / (rand.Float64() - 0.5)
	bounds := img.Bounds()
	start, end := findEdgeIntersections(bounds, p, m)
	line.Points = plotLine(start, end)
	lum := lineLuminosity(img, line)
	for i := 1; i < numTests; i++ {
		var newLine Line
		m = (rand.Float64() - 0.5) / (rand.Float64() - 0.5)
		newStart, newEnd := findEdgeIntersections(bounds, p, m)
		newLine.Points = plotLine(newStart, newEnd)
		newLum := lineLuminosity(img, newLine)
		if newLum/len(newLine.Points) < lum/len(line.Points) {
			line = newLine
			lum = newLum
		}
	}
	return line
}

func GenerateLines(img *image.Gray, lineCount, precision, lineWeight int, channel ColorChannel, out chan Line) {
	for i := 0; i < lineCount; i++ {
		darkestPixels := findDarkestPixels(img)
		selection := darkestPixels[rand.Intn(len(darkestPixels))]
		darkestLine := findDarkestLine(img, selection, precision)
		darkestLine.Weight = lineWeight
		darkestLine.Channel = channel
		if out != nil {
			out <- darkestLine
		}
		// Remove line from original image
		darkestLine.Weight = -lineWeight
		DrawLine(img, darkestLine, false)
	}
}

func TotalLuminosity(img *image.Gray) uint64 {
	var lum uint64
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			lum += uint64(img.Pix[y*img.Stride+x])
		}
	}
	return lum
}

func CalculateChannelRatios(redLum, greenLum, blueLum uint64, length int) (float64, float64, float64) {
	redLum = uint64(length*255) - redLum
	greenLum = uint64(length*255) - greenLum
	blueLum = uint64(length*255) - blueLum
	rLineRatio := float64(1.0)
	gLineRatio := float64(greenLum) / float64(redLum)
	bLineRatio := float64(blueLum) / float64(redLum)
	if greenLum > redLum && greenLum > blueLum {
		rLineRatio = float64(redLum) / float64(greenLum)
		gLineRatio = float64(1.0)
		bLineRatio = float64(blueLum) / float64(greenLum)
	} else if blueLum > redLum && blueLum > greenLum {
		rLineRatio = float64(redLum) / float64(blueLum)
		gLineRatio = float64(greenLum) / float64(blueLum)
		bLineRatio = float64(1.0)
	}
	return rLineRatio, gLineRatio, bLineRatio
}

func SetupFFMPEG(output string, width, height, fps int) (*exec.Cmd, io.WriteCloser, error) {
	exists := exec.Command("ffmpeg", "-version")
	if err := exists.Run(); err != nil {
		return nil, nil, err
	}
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%vx%v", width, height),
		"-r", fmt.Sprintf("%v", fps),
		"-i", "-",
		"-c:v", "libx264",
		"-crf", "18",
		"-preset", "veryslow",
		// "-pix_fmt", "yuv420p",
		output)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalln("Can't pipe ffmpeg stdin", err)
	}
	return cmd, stdin, nil
}
