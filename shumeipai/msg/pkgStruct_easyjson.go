// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package msg

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonD794903fDecodeShumeipaiMsg(in *jlexer.Lexer, out *SCPackageInfo) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "Error":
			out.Error = string(in.String())
		case "Result":
			out.Result = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD794903fEncodeShumeipaiMsg(out *jwriter.Writer, in SCPackageInfo) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"Error\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Error))
	}
	{
		const prefix string = ",\"Result\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Result))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v SCPackageInfo) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD794903fEncodeShumeipaiMsg(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v SCPackageInfo) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD794903fEncodeShumeipaiMsg(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *SCPackageInfo) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD794903fDecodeShumeipaiMsg(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *SCPackageInfo) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD794903fDecodeShumeipaiMsg(l, v)
}