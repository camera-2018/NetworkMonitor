package controller

type Resp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

var RespErrASNInvalid = Resp{
	Code: -1,
	Msg:  "invalid ASN",
}

var RespErrParamInvalid = Resp{
	Code: -2,
	Msg:  "invalid param",
}

var RespInternalError = Resp{
	Code: -3,
	Msg:  "internal error",
}

var RespErrASNNotFound = Resp{
	Code: -4,
	Msg:  "asn not found",
}

var RespErrNotEnabled = Resp{
	Code: -5,
	Msg:  "api not enabled",
}

var RespErrBGPGraphNotFound = Resp{
	Code: -6,
	Msg: "specified bgp graph not found",
}
