package source

type Literal struct {
	data any
	json []byte
}

func NewLiteral(v any, j []byte) *Literal {
	return &Literal{data: v, json: j}
}

func (L *Literal) Data() any    { return L.data }
func (L *Literal) Json() []byte { return L.json }
func (L *Literal) Unwrap() any  { return L.data }

type RawBytes struct {
	json []byte
}

func NewRawBytes(j []byte) *RawBytes {
	return &RawBytes{json: j}
}

func (R *RawBytes) Data() any    { return string(R.json) }
func (R *RawBytes) Json() []byte { return R.json }
