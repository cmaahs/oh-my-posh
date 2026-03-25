package segments

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"

	"github.com/jandedobbeleer/oh-my-posh/src/log"
	"github.com/jandedobbeleer/oh-my-posh/src/segments/options"
)

type Owm struct {
	Base

	FeelsLike   string
	Pressure    int
	Humidity    int
	Weather     string
	URL         string
	units       string
	UnitIcon    string
	Standard    string
	Imperial    string
	Metric      string
	Temperature int
}

const (
	// APIEnv environment variable that holds the openweathermap api key
	APIEnv options.Option = "apienv"
	// APIKey openweathermap api key
	APIKey options.Option = "api_key"
	// Location openweathermap location
	Location options.Option = "location"
	// Units openweathermap units
	Units options.Option = "units"
	// CacheKeyResponse key used when caching the response
	// CacheKeyResponse string = "owm_response"
	// CacheKeyURL key used when caching the url responsible for the response
	// CacheKeyURL string = "owm_url"
	// WithUnits is used to swith on an off the units on the individual measurements
	WithUnits options.Option = "with_units"

	ImperialIndicator = "°F"
	MetricIndicator   = "°C"
	StandardIndicator = "°K"
	// Environmental variable to dynamically set the Open Map API key
	OWMAPIKey string = "POSH_OWM_API_KEY"
	// Environmental variable to dynamically set the location string
	OWMLocationKey string = "POSH_OWM_LOCATION"
	CacheKeyURL    string = "owm_url"
)

type weather struct {
	ID               int    `json:"id"`
	ShortDescription string `json:"main"`
	Description      string `json:"description"`
	TypeID           string `json:"icon"`
}

// type temperature struct {
// 	Value     float64 `json:"temp"`
// 	FeelsLike float64 `json:"feels_like"`
// }

type main struct {
	Temp      float64 `json:"temp"`
	FeelsLike float64 `json:"feels_like"`
	TempMin   float64 `json:"temp_min"`
	TempMax   float64 `json:"temp_max"`
	Pressure  int     `json:"pressure"`
	Humidity  int     `json:"humidity"`
	SeaLevel  int     `json:"sea_level"`
	GrndLevel int     `json:"grnd_level"`
}

type owmDataResponse struct {
	Weather []weather `json:"weather"`
	Main    main      `json:"main"`
}

func (d *Owm) Enabled() bool {
	err := d.setStatus()

	if err != nil {
		log.Error(err)
		return false
	}

	return true
}

func (d *Owm) Template() string {
	return " {{ .Weather }} ({{ .Temperature }}{{ .UnitIcon }}) "
}

func (d *Owm) getResult() (*owmDataResponse, error) {
	response := new(owmDataResponse)

	apikey := d.options.Template(APIKey, "", d)
	if apikey == "" {
		return nil, errors.New("no api key found")
	}

	location := d.options.Template(Location, "", d)
	if location == "" {
		return nil, errors.New("no location found")
	}
	location = url.QueryEscape(location)

	apiEnv := d.options.String(APIEnv, "")

	units := d.options.String(Units, "standard")
	httpTimeout := d.options.Int(options.HTTPTimeout, options.DefaultHTTPTimeout)
	if apiEnv != "" {
		apikey, _ = os.LookupEnv(apiEnv)
	}

	d.URL = fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&units=%s&appid=%s", location, units, apikey)

	body, err := d.env.HTTPRequest(d.URL, nil, httpTimeout)
	if err != nil {
		return new(owmDataResponse), err
	}
	// fmt.Printf("%#v\n", body)

	err = json.Unmarshal(body, &response)
	if err != nil {
		return new(owmDataResponse), err
	}

	return response, nil
}

func (d *Owm) setStatus() error {
	units := d.options.String(Units, "standard")

	q, err := d.getResult()
	if err != nil {
		return err
	}

	if len(q.Weather) == 0 {
		return errors.New("no data found")
	}
	// id := q.Weather[0].TypeID
	wid := q.Weather[0].ID
	name := q.Weather[0].ShortDescription

	d.Temperature = int(math.Round(q.Main.Temp))
	d.Pressure = q.Main.Pressure
	d.Humidity = q.Main.Humidity
	d.FeelsLike = fmt.Sprintf("%d", int(math.Round(q.Main.FeelsLike)))
	icon := "☀️"
	// switch id {
	// case "01n":
	// 	fallthrough
	// case "01d":
	// 	icon = "\ufa98"
	// case "02n":
	// 	fallthrough
	// case "02d":
	// 	icon = "\ufa94"
	// case "03n":
	// 	fallthrough
	// case "03d":
	// 	icon = "\ue33d"
	// case "04n":
	// 	fallthrough
	// case "04d":
	// 	icon = "\ue312"
	// case "09n":
	// 	fallthrough
	// case "09d":
	// 	icon = "\ufa95"
	// case "10n":
	// 	fallthrough
	// case "10d":
	// 	icon = "\ue308"
	// case "11n":
	// 	fallthrough
	// case "11d":
	// 	icon = "\ue31d"
	// case "13n":
	// 	fallthrough
	// case "13d":
	// 	icon = "\ue31a"
	// case "50n":
	// 	fallthrough
	// case "50d":
	// 	icon = "\ue313"
	// }
	switch name {
	case "Thunderstorm":
		icon = "⛈"
	case "Drizzle":
		icon = "🌦"
	case "Rain":
		icon = "🌧"
	case "Snow":
		icon = "🌨"
	case "Tornado":
		icon = "🌪"
	case "Fog":
		icon = "💨"
	case "Clouds":
		if wid == 801 {
			icon = "️🌤"
		}
		if wid == 802 {
			icon = "⛅️"
		}
		if wid == 803 {
			icon = "🌥"
		}
		if wid == 804 {
			icon = "☁️"
		}
	}

	d.Weather = icon
	d.units = units
	d.UnitIcon = "\ue33e"

	withUnits := d.options.Bool(WithUnits, true)
	switch d.units {
	case "imperial":
		d.UnitIcon = ImperialIndicator // "°F" // \ue341"
		f := int(math.Round(q.Main.Temp))
		c := convertFahrenheitToCelsius(q.Main.Temp)
		k := convertFahrenheitToKelvin(q.Main.Temp)
		if withUnits {
			d.FeelsLike = fmt.Sprintf("%s%s", d.FeelsLike, ImperialIndicator)
			d.Imperial = fmt.Sprintf("%d%s", f, ImperialIndicator)
			d.Metric = fmt.Sprintf("%d%s", c, MetricIndicator)
			d.Standard = fmt.Sprintf("%d%s", k, StandardIndicator)
		} else {
			d.Imperial = fmt.Sprintf("%d", f)
			d.Metric = fmt.Sprintf("%d", c)
			d.Standard = fmt.Sprintf("%d", k)
		}
	case "metric":
		d.UnitIcon = MetricIndicator // "°C" // \ue339"
		c := int(math.Round(q.Main.Temp))
		f := convertCelsiusToFahrenheit(q.Main.Temp)
		k := convertCelsiusToKelvin(q.Main.Temp)
		if withUnits {
			d.FeelsLike = fmt.Sprintf("%s%s", d.FeelsLike, MetricIndicator)
			d.Imperial = fmt.Sprintf("%d%s", f, ImperialIndicator)
			d.Metric = fmt.Sprintf("%d%s", c, MetricIndicator)
			d.Standard = fmt.Sprintf("%d%s", k, StandardIndicator)
		} else {
			d.Imperial = fmt.Sprintf("%d", f)
			d.Metric = fmt.Sprintf("%d", c)
			d.Standard = fmt.Sprintf("%d", k)
		}
	case "":
		fallthrough
	case "standard":
		d.UnitIcon = StandardIndicator // "°K" // \ufa05"
		k := int(math.Round(q.Main.Temp))
		f := convertKelvinToFahrenheit(q.Main.Temp)
		c := convertKelvinToCelsius(q.Main.Temp)
		if withUnits {
			d.FeelsLike = fmt.Sprintf("%s%s", d.FeelsLike, StandardIndicator)
			d.Imperial = fmt.Sprintf("%d%s", f, ImperialIndicator)
			d.Metric = fmt.Sprintf("%d%s", c, MetricIndicator)
			d.Standard = fmt.Sprintf("%d%s", k, StandardIndicator)
		} else {
			d.Imperial = fmt.Sprintf("%d", f)
			d.Metric = fmt.Sprintf("%d", c)
			d.Standard = fmt.Sprintf("%d", k)
		}
	}
	return nil
}

func convertFahrenheitToCelsius(value float64) int {
	convertedValue := (value - 32) * 5.0 / 9.0
	return int(math.Round(convertedValue))
}

func convertCelsiusToFahrenheit(value float64) int {
	convertedValue := (value * 9.0 / 5.0) + 32
	return int(math.Round(convertedValue))
}

func convertFahrenheitToKelvin(value float64) int {
	//  F = 9/5(K - 273) + 32
	convertedValue := (9.0/5.0)*(value-273.15) + 32
	return int(math.Round(convertedValue))
}

func convertCelsiusToKelvin(value float64) int {
	convertedValue := value + 273.15
	return int(math.Round(convertedValue))
}

func convertKelvinToFahrenheit(value float64) int {
	// K = 5/9(F - 32) + 273.15
	convertedValue := 5.0/9.0*(value-32) + 273.15
	return int(math.Round(convertedValue))
}

func convertKelvinToCelsius(value float64) int {
	convertedValue := value - 273.15
	return int(math.Round(convertedValue))
}

// func (d *Owm) Init(props properties.Properties, env platform.Environment) {
// 	d.props = props
// 	d.env = env
// }
