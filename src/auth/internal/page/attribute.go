package page

type Attribute struct {
	Key   string
	Value string
}

func NewAttribute(key string, value string) Attribute {
	return Attribute{
		Key:   key,
		Value: value,
	}
}

func NewAttributes(pairs ...string) []Attribute {
	attributes := make([]Attribute, 0, len(pairs)/2)
	for i := 0; i < len(pairs)-1; i += 2 {
		attributes = append(attributes, NewAttribute(pairs[i], pairs[i+1]))
	}
	return attributes
}
