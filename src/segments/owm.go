package segments

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"

	"github.com/jandedobbeleer/oh-my-posh/src/log"
	"github.com/jandedobbeleer/oh-my-posh/src/platform"
	"github.com/jandedobbeleer/oh-my-posh/src/properties"
)

type Owm struct {
	base

	FeelsLike   string
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
	APIEnv properties.Property = "apienv"
	// APIKey openweathermap api key
	APIKey properties.Property = "api_key"
	// Location openweathermap location
	Location properties.Property = "location"
	// Units openweathermap units
	Units properties.Property = "units"
	// CacheKeyResponse key used when caching the response
	CacheKeyResponse string = "owm_response"
	// CacheKeyURL key used when caching the url responsible for the response
	CacheKeyURL string = "owm_url"
	// WithUnits is used to swith on an off the units on the individual measurements
	WithUnits properties.Property = "with_units"

	ImperialIndicator = "°F"
	MetricIndicator   = "°C"
	StandardIndicator = "°K"
	PoshOWMAPIKey     = "POSH_OWM_API_KEY"
)

type weather struct {
	ID               int    `json:"id"`
	ShortDescription string `json:"main"`
	Description      string `json:"description"`
	TypeID           string `json:"icon"`
}

type temperature struct {
	Value     float64 `json:"temp"`
	FeelsLike float64 `json:"feels_like"`
}

type owmDataResponse struct {
	Data        []weather `json:"weather"`
	temperature `json:"main"`
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

	apikey := properties.OneOf(d.props, ".", APIKey, "apiKey")
	if len(apikey) == 0 {
		apikey = d.env.Getenv(PoshOWMAPIKey)
	}

	apiEnv := d.props.GetString(APIEnv, "")
	// apikey := d.props.GetString(APIKey, ".")
	location := d.props.GetString(Location, "De Bilt,NL")
	location = url.QueryEscape(location)

	if len(apikey) == 0 || len(location) == 0 {
		return nil, errors.New("no api key or location found")
	}

	units := d.props.GetString(Units, "standard")
	httpTimeout := d.props.GetInt(properties.HTTPTimeout, properties.DefaultHTTPTimeout)
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
	units := d.props.GetString(Units, "standard")

	q, err := d.getResult()
	if err != nil {
		return err
	}

	if len(q.Data) == 0 {
		return errors.New("No data found")
	}
	// id := q.Data[0].TypeID
	wid := q.Data[0].ID
	name := q.Data[0].ShortDescription

	d.Temperature = int(math.Round(q.temperature.Value))
	d.FeelsLike = fmt.Sprintf("%d", int(math.Round(q.temperature.FeelsLike)))
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

	withUnits := d.props.GetBool(WithUnits, true)
	switch d.units {
	case "imperial":
		d.UnitIcon = ImperialIndicator // "°F" // \ue341"
		f := int(math.Round(q.temperature.Value))
		c := convertFahrenheitToCelsius(q.temperature.Value)
		k := convertFahrenheitToKelvin(q.temperature.Value)
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
		c := int(math.Round(q.temperature.Value))
		f := convertCelsiusToFahrenheit(q.temperature.Value)
		k := convertCelsiusToKelvin(q.temperature.Value)
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
		k := int(math.Round(q.temperature.Value))
		f := convertKelvinToFahrenheit(q.temperature.Value)
		c := convertKelvinToCelsius(q.temperature.Value)
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

func (d *Owm) Init(props properties.Properties, env platform.Environment) {
	d.props = props
	d.env = env
}
