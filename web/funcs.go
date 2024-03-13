package web

import (
	"errors"
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"

	"github.com/ttocsneb/station-webapp/util"
)

func safe_html(html string) template.HTML {
	return template.HTML(html)
}

func embed_func(name string, args ...any) (template.HTML, error) {
	fn, exists := embedFuncs[name]
	if !exists {
		return "", fmt.Errorf("embed %v does not exist", name)
	}
	buf := util.BufPool.Get()
	defer util.BufPool.Put(buf)
	err := fn(buf, args)
	if err != nil {
		return "", err
	}
	output := template.HTML(buf.String())
	return output, nil
}

func make_dict(args ...any) (map[string]any, error) {
	dict := make(map[string]any)
	i := 0
	for i < len(args) {
		key, exists := args[i].(string)
		if !exists {
			return nil, errors.New("Key must be a string")
		}
		i += 1

		val := args[i]
		dict[key] = val
		i += 1
	}

	return dict, nil
}

func cardinal_angle(angle any) (string, error) {
	rot, exists := angle.(int)
	if !exists {
		flt, exists := angle.(float64)
		if !exists {
			return "", errors.New("Angle must be a number")
		}
		rot = int(flt)
	}

	var text string
	rot = rot % 360
	if rot < 12 || rot > 348 {
		text = "N"
	} else if rot < 34 {
		text = "NNE"
	} else if rot < 57 {
		text = "NE"
	} else if rot < 79 {
		text = "ENE"
	} else if rot < 102 {
		text = "E"
	} else if rot < 124 {
		text = "ESE"
	} else if rot < 147 {
		text = "SE"
	} else if rot < 169 {
		text = "SSE"
	} else if rot < 192 {
		text = "S"
	} else if rot < 214 {
		text = "SSW"
	} else if rot < 237 {
		text = "SW"
	} else if rot < 259 {
		text = "WSW"
	} else if rot < 282 {
		text = "W"
	} else if rot < 304 {
		text = "WNW"
	} else if rot < 327 {
		text = "NW"
	} else {
		text = "NNW"
	}
	return text, nil
}

func cardinal_angle_aria(angle any) (string, error) {
	text, err := cardinal_angle(angle)
	if err != nil {
		return text, err
	}

	labels := make([]string, len(text))
	for i, c := range text {
		switch c {
		case 'N':
			labels[i] = "North"
		case 'W':
			labels[i] = "West"
		case 'E':
			labels[i] = "East"
		case 'S':
			labels[i] = "South"
		}
	}

	return strings.Join(labels, " "), nil
}

func round(val float64) int {
	return int(math.Round(val))
}

func round_nth(val float64, precision int) float64 {
	return math.Round(val*math.Pow10(precision)) / math.Pow10(precision)
}

func ftime(t time.Time, format string) string {
	switch format {
	case "Layout":
		format = time.Layout
	case "ANSIC":
		format = time.ANSIC
	case "UnixDate":
		format = time.UnixDate
	case "RubyDate":
		format = time.RubyDate
	case "RFC822":
		format = time.RFC822
	case "RFC822Z":
		format = time.RFC822Z
	case "RFC850":
		format = time.RFC850
	case "RFC1123":
		format = time.RFC1123
	case "RFC1123Z":
		format = time.RFC1123Z
	case "RFC3339":
		format = time.RFC3339
	case "RFC3339Nano":
		format = time.RFC3339Nano
	case "Kitchen":
		format = time.Kitchen
	case "Stamp":
		format = time.Stamp
	case "StampMilli":
		format = time.StampMilli
	case "StampMicro":
		format = time.StampMicro
	case "StampNano":
		format = time.StampNano
	case "DateTime":
		format = time.DateTime
	case "DateOnly":
		format = time.DateOnly
	case "TimeOnly":
		format = time.TimeOnly
	}
	return t.Format(format)
}

const IMPERIAL = "imperial"
const METRIC = "metric"
const MIXED = "mixed"

func convertTemp(value float64, unit string, system string) (float64, string) {
	if unit == "K" {
		value = value - 272.15
		unit = "C"
	}
	if unit == "C" && system == IMPERIAL {
		value = value*9/5 + 32
		unit = "F"
	}
	if unit == "F" && (system == METRIC || system == MIXED) {
		value = (value - 32) * 5 / 9
		unit = "C"
	}

	return float64(round(value)), unit
}

// barom: 858.05
// dailyrain: 0
// dewpoint: -6.344679
// humidity: 44.08316
// rain-1h: 0
// temp: 4.8760285
// uv: 0
// winddir: 270
// winddir-avg10m: 99
// winddir-avg2m: 277
// windgustdir-2m: 270
// windgustspd-2m: 1.0054344
// windspd: 1.0054344
// windspd-avg10m: 0.77418447
// windspd-avg2m: 0.3267662

func convertPressure(value float64, unit string, system string) (float64, string) {
	if unit == "Pa" {
		value = value / 100
		unit = "hPa"
	}
	if unit == "kPa" {
		value = value * 10
		unit = "hPa"
	}
	if unit == "hPa" && (system == IMPERIAL || system == MIXED) {
		value = value / 33.86388666666671
		unit = "inHg"
	}
	if unit == "inHg" && system == METRIC {
		value = value * 33.86388666666671
		unit = "hPa"
	}

	if unit == "hPa" {
		value = float64(round(value))
	} else if unit == "inHg" {
		value = round_nth(value, 2)
	}

	return value, unit
}

func convertRain(value float64, unit string, system string) (float64, string) {
	if unit == "in" && system == METRIC {
		value = value * 25.4
		unit = "mm"
	}
	if unit == "mm" && (system == IMPERIAL || system == MIXED) {
		value = value / 25.4
		unit = "in"
	}

	if unit == "mm" {
		value = round_nth(value, 1)
	} else if unit == "in" {
		value = round_nth(value, 2)
	}

	return value, unit
}

func convertSpeed(value float64, unit string, system string) (float64, string) {
	if unit == "m/s" {
		unit = "km/h"
		value = value * 3.6
	}
	if unit == "km/h" && (system == IMPERIAL || system == MIXED) {
		value = value / 1.609344
		unit = "mph"
	}
	if unit == "mph" && (system == METRIC) {
		value = value * 1.609344
		unit = "km/h"
	}

	return float64(round(value)), unit
}

func convert(value float64, unit string, sensor string, system string) (float64, string) {
	switch sensor {
	case "temp":
		return convertTemp(value, unit, system)
	case "rain":
		return convertRain(value, unit, system)
	case "pressure":
		return convertPressure(value, unit, system)
	case "speed":
		return convertSpeed(value, unit, system)
	}

	return value, unit
}

func get_unit(value float64, unit string, sensor string, system string) string {
	value, unit = convert(value, unit, sensor, system)
	return unit
}
func get_value(value float64, unit string, sensor string, system string) float64 {
	value, unit = convert(value, unit, sensor, system)
	return value
}

func route(path string) string {
	return fmt.Sprintf("%v%v", util.Conf.Base, path)
}

var funcs = template.FuncMap{
	"safe":                safe_html,
	"embed":               embed_func,
	"dict":                make_dict,
	"cardinal_angle":      cardinal_angle,
	"cardinal_angle_aria": cardinal_angle_aria,
	"round":               round,
	"round_nth":           round_nth,
	"ftime":               ftime,
	"convert":             get_value,
	"get_unit":            get_unit,
	"route":               route,
}
