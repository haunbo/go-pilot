package xstring

import "encoding/json"

// Json marshals obj and ignores errors.
func Json(obj any) string {
	aa, _ := json.Marshal(obj)
	return string(aa)
}

// JsonBytes marshals obj and ignores errors.
func JsonBytes(obj any) []byte {
	aa, _ := json.Marshal(obj)
	return aa
}

// PrettyJson marshals obj with indentation and ignores errors.
func PrettyJson(obj any) string {
	aa, _ := json.MarshalIndent(obj, "", "    ")
	return string(aa)
}
