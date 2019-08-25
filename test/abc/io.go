package abc

type TestReq struct {
	Id    string            `kit_url:"id"`
	Query string            `kit_query:"query"`
	Body  map[string]string `kit_body:"query" kit_encoder:"json"`
	Name  string            `json:"name"`
}
type TestResp struct {
	Testing string `json:"testing"`
}
