package segments

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/jandedobbeleer/oh-my-posh/src/platform"
	"github.com/jandedobbeleer/oh-my-posh/src/properties"
)

type Owm struct {
	props properties.Properties
	env   platform.Environment

	Temperature int
	FeelsLike   string
	Weather     string
	URL         string
	units       string
	UnitIcon    string
	Standard    string
	Imperial    string
	Metric      string
}

const (
	// APIEnv environment variable that holds the openweathermap api key
	APIEnv properties.Property = "apienv"
	// APIKey openweathermap api key
	APIKey properties.Property = "apikey"
	// Location openweathermap location
	Location properties.Property = "location"
	// Units openweathermap units
	Units properties.Property = "units"
	// Latitude for the location used in place of location
	Latitude properties.Property = "latitude"
	// Longitude for the location used in place of location
	Longitude properties.Property = "longitude"
	// CacheKeyResponse key used when caching the response
	CacheKeyResponse string = "owm_response"
	// CacheKeyURL key used when caching the url responsible for the response
	CacheKeyURL string = "owm_url"
	// WithUnits is used to swith on an off the units on the individual measurements
	WithUnits properties.Property = "with_units"

	ImperialIndicator = "°F"
	MetricIndicator   = "°C"
	StandardIndicator = "°K"
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

type geoLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func (d *Owm) Enabled() bool {
	err := d.setStatus()

	if err != nil {
		d.env.Error(err)
		return false
	}

	return true
}

func (d *Owm) Template() string {
	return " {{ .Weather }} ({{ .Temperature }}{{ .UnitIcon }}) "
}

func (d *Owm) getResult() (*owmDataResponse, error) {
	cacheTimeout := d.props.GetInt(properties.CacheTimeout, properties.DefaultCacheTimeout)
	response := new(owmDataResponse)
	if cacheTimeout > 0 {
		// check if data stored in cache
		val, found := d.env.Cache().Get(CacheKeyResponse)
		// we got something from te cache
		if found {
			err := json.Unmarshal([]byte(val), response)
			if err != nil {
				return nil, err
			}
			d.URL, _ = d.env.Cache().Get(CacheKeyURL)
			return response, nil
		}
	}

	apiEnv := d.props.GetString(APIEnv, "")
	apikey := d.props.GetString(APIKey, ".")
	location := d.props.GetString(Location, "De Bilt,NL")
	latitude := d.props.GetFloat64(Latitude, 91)    // This default value is intentionally invalid since there should not be a default for this and 0 is a valid value
	longitude := d.props.GetFloat64(Longitude, 181) // This default value is intentionally invalid since there should not be a default for this and 0 is a valid value
	units := d.props.GetString(Units, "standard")
	httpTimeout := d.props.GetInt(properties.HTTPTimeout, properties.DefaultHTTPTimeout)
	if apiEnv != "" {
		apikey, _ = os.LookupEnv(apiEnv)
	}

	validCoordinates := func(latitude, longitude float64) bool {
		// Latitude values are only valid if they are between -90 and 90
		// Longitude values are only valid if they are between -180 and 180
		// https://gisgeography.com/latitude-longitude-coordinates/
		return latitude <= 90 && latitude >= -90 && longitude <= 180 && longitude >= -180
	}

	if !validCoordinates(latitude, longitude) {
		var geoResponse []geoLocation
		geocodingURL := fmt.Sprintf("http://api.openweathermap.org/geo/1.0/direct?q=%s&limit=1&appid=%s", location, apikey)

		body, err := d.env.HTTPRequest(geocodingURL, nil, httpTimeout)
		if err != nil {
			return new(owmDataResponse), err
		}

		err = json.Unmarshal(body, &geoResponse)
		if err != nil {
			return new(owmDataResponse), err
		}

		if len(geoResponse) == 0 {
			return new(owmDataResponse), fmt.Errorf("no coordinates found for %s", location)
		}

		latitude = geoResponse[0].Lat
		longitude = geoResponse[0].Lon
	}

	d.URL = fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?lat=%v&lon=%v&units=%s&appid=%s", latitude, longitude, units, apikey)

	body, err := d.env.HTTPRequest(d.URL, nil, httpTimeout)
	if err != nil {
		return new(owmDataResponse), err
	}
	// fmt.Printf("%#v\n", body)
	err = json.Unmarshal(body, &response)
	if err != nil {
		return new(owmDataResponse), err
	}

	if cacheTimeout > 0 {
		// persist new forecasts in cache
		d.env.Cache().Set(CacheKeyResponse, string(body), cacheTimeout)
		d.env.Cache().Set(CacheKeyURL, d.URL, cacheTimeout)
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
