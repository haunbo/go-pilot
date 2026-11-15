package conf

// Unmarshaller decodes raw config bytes into a map or struct.
type Unmarshaller = func([]byte, any) error

// GetOption customizes UnmarshalKey behavior.
type GetOption func(*GetOptions)

// GetOptions holds UnmarshalKey options.
type GetOptions struct {
	TagName string
}

var defaultGetOptions = GetOptions{TagName: "json"}

// TagName sets the struct tag used when decoding config into a struct.
func TagName(tag string) GetOption {
	return func(o *GetOptions) {
		o.TagName = tag
	}
}

// TagNameJSON decodes with json struct tags.
func TagNameJSON() GetOption { return TagName("json") }

// TagNameYAML decodes with yaml struct tags.
func TagNameYAML() GetOption { return TagName("yaml") }

// TagNameMapstructure decodes with mapstructure struct tags.
func TagNameMapstructure() GetOption { return TagName("mapstructure") }
