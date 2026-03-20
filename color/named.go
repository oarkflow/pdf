package color

import (
	"fmt"
	"strings"
)

var (
	AliceBlue            = FromRGBi(240, 248, 255)
	AntiqueWhite         = FromRGBi(250, 235, 215)
	Aqua                 = FromRGBi(0, 255, 255)
	Aquamarine           = FromRGBi(127, 255, 212)
	Azure                = FromRGBi(240, 255, 255)
	Beige                = FromRGBi(245, 245, 220)
	Bisque               = FromRGBi(255, 228, 196)
	Black                = RGB{0, 0, 0}
	BlanchedAlmond       = FromRGBi(255, 235, 205)
	Blue                 = RGB{0, 0, 1}
	BlueViolet           = FromRGBi(138, 43, 226)
	Brown                = FromRGBi(165, 42, 42)
	BurlyWood            = FromRGBi(222, 184, 135)
	CadetBlue            = FromRGBi(95, 158, 160)
	Chartreuse           = FromRGBi(127, 255, 0)
	Chocolate            = FromRGBi(210, 105, 30)
	Coral                = FromRGBi(255, 127, 80)
	CornflowerBlue       = FromRGBi(100, 149, 237)
	Cornsilk             = FromRGBi(255, 248, 220)
	Crimson              = FromRGBi(220, 20, 60)
	Cyan                 = FromRGBi(0, 255, 255)
	DarkBlue             = FromRGBi(0, 0, 139)
	DarkCyan             = FromRGBi(0, 139, 139)
	DarkGoldenrod        = FromRGBi(184, 134, 11)
	DarkGray             = FromRGBi(169, 169, 169)
	DarkGreen            = FromRGBi(0, 100, 0)
	DarkKhaki            = FromRGBi(189, 183, 107)
	DarkMagenta          = FromRGBi(139, 0, 139)
	DarkOliveGreen       = FromRGBi(85, 107, 47)
	DarkOrange           = FromRGBi(255, 140, 0)
	DarkOrchid           = FromRGBi(153, 50, 204)
	DarkRed              = FromRGBi(139, 0, 0)
	DarkSalmon           = FromRGBi(233, 150, 122)
	DarkSeaGreen         = FromRGBi(143, 188, 143)
	DarkSlateBlue        = FromRGBi(72, 61, 139)
	DarkSlateGray        = FromRGBi(47, 79, 79)
	DarkTurquoise        = FromRGBi(0, 206, 209)
	DarkViolet           = FromRGBi(148, 0, 211)
	DeepPink             = FromRGBi(255, 20, 147)
	DeepSkyBlue          = FromRGBi(0, 191, 255)
	DimGray              = FromRGBi(105, 105, 105)
	DodgerBlue           = FromRGBi(30, 144, 255)
	Firebrick            = FromRGBi(178, 34, 34)
	FloralWhite          = FromRGBi(255, 250, 240)
	ForestGreen          = FromRGBi(34, 139, 34)
	Fuchsia              = FromRGBi(255, 0, 255)
	Gainsboro            = FromRGBi(220, 220, 220)
	GhostWhite           = FromRGBi(248, 248, 255)
	Gold                 = FromRGBi(255, 215, 0)
	Goldenrod            = FromRGBi(218, 165, 32)
	GrayColor            = FromRGBi(128, 128, 128)
	Green                = FromRGBi(0, 128, 0)
	GreenYellow          = FromRGBi(173, 255, 47)
	Honeydew             = FromRGBi(240, 255, 240)
	HotPink              = FromRGBi(255, 105, 180)
	IndianRed            = FromRGBi(205, 92, 92)
	Indigo               = FromRGBi(75, 0, 130)
	Ivory                = FromRGBi(255, 255, 240)
	Khaki                = FromRGBi(240, 230, 140)
	Lavender             = FromRGBi(230, 230, 250)
	LavenderBlush        = FromRGBi(255, 240, 245)
	LawnGreen            = FromRGBi(124, 252, 0)
	LemonChiffon         = FromRGBi(255, 250, 205)
	LightBlue            = FromRGBi(173, 216, 230)
	LightCoral           = FromRGBi(240, 128, 128)
	LightCyan            = FromRGBi(224, 255, 255)
	LightGoldenrodYellow = FromRGBi(250, 250, 210)
	LightGray            = FromRGBi(211, 211, 211)
	LightGreen           = FromRGBi(144, 238, 144)
	LightPink            = FromRGBi(255, 182, 193)
	LightSalmon          = FromRGBi(255, 160, 122)
	LightSeaGreen        = FromRGBi(32, 178, 170)
	LightSkyBlue         = FromRGBi(135, 206, 250)
	LightSlateGray       = FromRGBi(119, 136, 153)
	LightSteelBlue       = FromRGBi(176, 196, 222)
	LightYellow          = FromRGBi(255, 255, 224)
	Lime                 = FromRGBi(0, 255, 0)
	LimeGreen            = FromRGBi(50, 205, 50)
	Linen                = FromRGBi(250, 240, 230)
	Magenta              = FromRGBi(255, 0, 255)
	Maroon               = FromRGBi(128, 0, 0)
	MediumAquamarine     = FromRGBi(102, 205, 170)
	MediumBlue           = FromRGBi(0, 0, 205)
	MediumOrchid         = FromRGBi(186, 85, 211)
	MediumPurple         = FromRGBi(147, 112, 219)
	MediumSeaGreen       = FromRGBi(60, 179, 113)
	MediumSlateBlue      = FromRGBi(123, 104, 238)
	MediumSpringGreen    = FromRGBi(0, 250, 154)
	MediumTurquoise      = FromRGBi(72, 209, 204)
	MediumVioletRed      = FromRGBi(199, 21, 133)
	MidnightBlue         = FromRGBi(25, 25, 112)
	MintCream            = FromRGBi(245, 255, 250)
	MistyRose            = FromRGBi(255, 228, 225)
	Moccasin             = FromRGBi(255, 228, 181)
	NavajoWhite          = FromRGBi(255, 222, 173)
	Navy                 = FromRGBi(0, 0, 128)
	OldLace              = FromRGBi(253, 245, 230)
	Olive                = FromRGBi(128, 128, 0)
	OliveDrab            = FromRGBi(107, 142, 35)
	Orange               = FromRGBi(255, 165, 0)
	OrangeRed            = FromRGBi(255, 69, 0)
	Orchid               = FromRGBi(218, 112, 214)
	PaleGoldenrod        = FromRGBi(238, 232, 170)
	PaleGreen            = FromRGBi(152, 251, 152)
	PaleTurquoise        = FromRGBi(175, 238, 238)
	PaleVioletRed        = FromRGBi(219, 112, 147)
	PapayaWhip           = FromRGBi(255, 239, 213)
	PeachPuff            = FromRGBi(255, 218, 185)
	Peru                 = FromRGBi(205, 133, 63)
	Pink                 = FromRGBi(255, 192, 203)
	Plum                 = FromRGBi(221, 160, 221)
	PowderBlue           = FromRGBi(176, 224, 230)
	Purple               = FromRGBi(128, 0, 128)
	RebeccaPurple        = FromRGBi(102, 51, 153)
	Red                  = RGB{1, 0, 0}
	RosyBrown            = FromRGBi(188, 143, 143)
	RoyalBlue            = FromRGBi(65, 105, 225)
	SaddleBrown          = FromRGBi(139, 69, 19)
	Salmon               = FromRGBi(250, 128, 114)
	SandyBrown           = FromRGBi(244, 164, 96)
	SeaGreen             = FromRGBi(46, 139, 87)
	Seashell             = FromRGBi(255, 245, 238)
	Sienna               = FromRGBi(160, 82, 45)
	Silver               = FromRGBi(192, 192, 192)
	SkyBlue              = FromRGBi(135, 206, 235)
	SlateBlue            = FromRGBi(106, 90, 205)
	SlateGray            = FromRGBi(112, 128, 144)
	Snow                 = FromRGBi(255, 250, 250)
	SpringGreen          = FromRGBi(0, 255, 127)
	SteelBlue            = FromRGBi(70, 130, 180)
	Tan                  = FromRGBi(210, 180, 140)
	Teal                 = FromRGBi(0, 128, 128)
	Thistle              = FromRGBi(216, 191, 216)
	Tomato               = FromRGBi(255, 99, 71)
	Turquoise            = FromRGBi(64, 224, 208)
	Violet               = FromRGBi(238, 130, 238)
	Wheat                = FromRGBi(245, 222, 179)
	White                = RGB{1, 1, 1}
	WhiteSmoke           = FromRGBi(245, 245, 245)
	Yellow               = FromRGBi(255, 255, 0)
	YellowGreen          = FromRGBi(154, 205, 50)
)

var namedColors map[string]RGB

func init() {
	namedColors = map[string]RGB{
		"aliceblue":            AliceBlue,
		"antiquewhite":         AntiqueWhite,
		"aqua":                 Aqua,
		"aquamarine":           Aquamarine,
		"azure":                Azure,
		"beige":                Beige,
		"bisque":               Bisque,
		"black":                Black,
		"blanchedalmond":       BlanchedAlmond,
		"blue":                 Blue,
		"blueviolet":           BlueViolet,
		"brown":                Brown,
		"burlywood":            BurlyWood,
		"cadetblue":            CadetBlue,
		"chartreuse":           Chartreuse,
		"chocolate":            Chocolate,
		"coral":                Coral,
		"cornflowerblue":       CornflowerBlue,
		"cornsilk":             Cornsilk,
		"crimson":              Crimson,
		"cyan":                 Cyan,
		"darkblue":             DarkBlue,
		"darkcyan":             DarkCyan,
		"darkgoldenrod":        DarkGoldenrod,
		"darkgray":             DarkGray,
		"darkgreen":            DarkGreen,
		"darkkhaki":            DarkKhaki,
		"darkmagenta":          DarkMagenta,
		"darkolivegreen":       DarkOliveGreen,
		"darkorange":           DarkOrange,
		"darkorchid":           DarkOrchid,
		"darkred":              DarkRed,
		"darksalmon":           DarkSalmon,
		"darkseagreen":        DarkSeaGreen,
		"darkslateblue":        DarkSlateBlue,
		"darkslategray":        DarkSlateGray,
		"darkturquoise":        DarkTurquoise,
		"darkviolet":           DarkViolet,
		"deeppink":             DeepPink,
		"deepskyblue":          DeepSkyBlue,
		"dimgray":              DimGray,
		"dodgerblue":           DodgerBlue,
		"firebrick":            Firebrick,
		"floralwhite":          FloralWhite,
		"forestgreen":          ForestGreen,
		"fuchsia":              Fuchsia,
		"gainsboro":            Gainsboro,
		"ghostwhite":           GhostWhite,
		"gold":                 Gold,
		"goldenrod":            Goldenrod,
		"gray":                 GrayColor,
		"green":                Green,
		"greenyellow":          GreenYellow,
		"honeydew":             Honeydew,
		"hotpink":              HotPink,
		"indianred":            IndianRed,
		"indigo":               Indigo,
		"ivory":                Ivory,
		"khaki":                Khaki,
		"lavender":             Lavender,
		"lavenderblush":        LavenderBlush,
		"lawngreen":            LawnGreen,
		"lemonchiffon":         LemonChiffon,
		"lightblue":            LightBlue,
		"lightcoral":           LightCoral,
		"lightcyan":            LightCyan,
		"lightgoldenrodyellow": LightGoldenrodYellow,
		"lightgray":            LightGray,
		"lightgreen":           LightGreen,
		"lightpink":            LightPink,
		"lightsalmon":          LightSalmon,
		"lightseagreen":        LightSeaGreen,
		"lightskyblue":         LightSkyBlue,
		"lightslategray":       LightSlateGray,
		"lightsteelblue":       LightSteelBlue,
		"lightyellow":          LightYellow,
		"lime":                 Lime,
		"limegreen":            LimeGreen,
		"linen":                Linen,
		"magenta":              Magenta,
		"maroon":               Maroon,
		"mediumaquamarine":     MediumAquamarine,
		"mediumblue":           MediumBlue,
		"mediumorchid":         MediumOrchid,
		"mediumpurple":         MediumPurple,
		"mediumseagreen":       MediumSeaGreen,
		"mediumslateblue":      MediumSlateBlue,
		"mediumspringgreen":    MediumSpringGreen,
		"mediumturquoise":      MediumTurquoise,
		"mediumvioletred":      MediumVioletRed,
		"midnightblue":         MidnightBlue,
		"mintcream":            MintCream,
		"mistyrose":            MistyRose,
		"moccasin":             Moccasin,
		"navajowhite":          NavajoWhite,
		"navy":                 Navy,
		"oldlace":              OldLace,
		"olive":                Olive,
		"olivedrab":            OliveDrab,
		"orange":               Orange,
		"orangered":            OrangeRed,
		"orchid":               Orchid,
		"palegoldenrod":        PaleGoldenrod,
		"palegreen":            PaleGreen,
		"paleturquoise":        PaleTurquoise,
		"palevioletred":        PaleVioletRed,
		"papayawhip":           PapayaWhip,
		"peachpuff":            PeachPuff,
		"peru":                 Peru,
		"pink":                 Pink,
		"plum":                 Plum,
		"powderblue":           PowderBlue,
		"purple":               Purple,
		"rebeccapurple":        RebeccaPurple,
		"red":                  Red,
		"rosybrown":            RosyBrown,
		"royalblue":            RoyalBlue,
		"saddlebrown":          SaddleBrown,
		"salmon":               Salmon,
		"sandybrown":           SandyBrown,
		"seagreen":             SeaGreen,
		"seashell":             Seashell,
		"sienna":               Sienna,
		"silver":               Silver,
		"skyblue":              SkyBlue,
		"slateblue":            SlateBlue,
		"slategray":            SlateGray,
		"snow":                 Snow,
		"springgreen":          SpringGreen,
		"steelblue":            SteelBlue,
		"tan":                  Tan,
		"teal":                 Teal,
		"thistle":              Thistle,
		"tomato":               Tomato,
		"turquoise":            Turquoise,
		"violet":               Violet,
		"wheat":                Wheat,
		"white":                White,
		"whitesmoke":           WhiteSmoke,
		"yellow":               Yellow,
		"yellowgreen":          YellowGreen,
	}
}

// ByName returns a named CSS color or an error if not found.
func ByName(name string) (RGB, error) {
	c, ok := namedColors[strings.ToLower(name)]
	if !ok {
		return RGB{}, fmt.Errorf("color: unknown color name %q", name)
	}
	return c, nil
}
