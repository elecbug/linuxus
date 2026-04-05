package page

type KeyValue struct {
	Key   string
	Value string
}

func NewKeyValue(key string, value string) KeyValue {
	return KeyValue{
		Key:   key,
		Value: value,
	}
}
