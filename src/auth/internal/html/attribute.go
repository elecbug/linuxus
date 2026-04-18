package html

// Attribute represents a key-value attribute in HTML or CSS rendering.
type Attribute struct {
	// Key is the attribute name.
	Key string
	// Value is the attribute value.
	Value string
}

// NewAttribute creates a single attribute from key and value.
func NewAttribute(key string, value string) Attribute {
	return Attribute{
		Key:   key,
		Value: value,
	}
}

// NewAttributes creates attributes from alternating key/value pairs.
func NewAttributes(pairs ...string) []Attribute {
	attributes := make([]Attribute, 0, len(pairs)/2)
	for i := 0; i < len(pairs)-1; i += 2 {
		attributes = append(attributes, NewAttribute(pairs[i], pairs[i+1]))
	}
	return attributes
}
