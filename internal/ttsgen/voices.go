package ttsgen

// Voice 表示一个可选的 TTS 音色。
type Voice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// VoicesForProvider 返回指定 provider 的已知音色列表。
// 未知 provider 返回 nil，调用方可据此回退到默认音色输入框。
func VoicesForProvider(provider string) []Voice {
	switch provider {
	case "stepfun":
		return stepFunVoices
	case "openai":
		return openaiVoices
	default:
		return nil
	}
}

var stepFunVoices = []Voice{
	{ID: "cixingnansheng", Name: "磁性男声"},
	{ID: "cixinvsheng", Name: "磁性女声"},
	{ID: "wenrounansheng", Name: "温柔男声"},
	{ID: "wenrounvsheng", Name: "温柔女声"},
	{ID: "wenrougongzi", Name: "温柔公子"},
	{ID: "wenroushunv", Name: "温柔熟女"},
	{ID: "tianmeinvsheng", Name: "甜美女声"},
	{ID: "qingchunshaonv", Name: "清纯少女"},
	{ID: "yuanqishaonv", Name: "元气少女"},
	{ID: "yuanqinansheng", Name: "元气男声"},
	{ID: "ruyananshi", Name: "儒雅男士"},
	{ID: "zhengpaiqingnian", Name: "正派青年"},
	{ID: "jingdiannvsheng", Name: "经典女声"},
	{ID: "qinhenvsheng", Name: "亲和女声"},
	{ID: "huolinvsheng", Name: "活力女声"},
	{ID: "shuangkuainansheng", Name: "爽快男声"},
	{ID: "ganliannvsheng", Name: "干练女声"},
	{ID: "boyinnansheng", Name: "播音男声"},
	{ID: "shenchennanyin", Name: "深沉男音"},
	{ID: "qinqienvsheng", Name: "亲切女声"},
	{ID: "linjiajiejie", Name: "邻家姐姐"},
	{ID: "linjiameimei", Name: "邻家妹妹"},
	{ID: "ruanmengnvsheng", Name: "软萌女声"},
	{ID: "jilingshaonv", Name: "机灵少女"},
	{ID: "zhixingjiejie", Name: "知性姐姐"},
	{ID: "qingniandaxuesheng", Name: "青年大学生"},
	{ID: "youyanvsheng", Name: "优雅女声"},
	{ID: "lengyanyujie", Name: "冷艳御姐"},
	{ID: "shuangkuaijiejie", Name: "爽快姐姐"},
	{ID: "wenjingxuejie", Name: "文静学姐"},
	{ID: "elegantgentle-female", Name: "气质温婉"},
	{ID: "livelybreezy-female", Name: "活力轻快"},
}

var openaiVoices = []Voice{
	{ID: "alloy", Name: "Alloy"},
	{ID: "echo", Name: "Echo"},
	{ID: "fable", Name: "Fable"},
	{ID: "onyx", Name: "Onyx"},
	{ID: "nova", Name: "Nova"},
	{ID: "shimmer", Name: "Shimmer"},
	{ID: "coral", Name: "Coral"},
	{ID: "sage", Name: "Sage"},
	{ID: "ash", Name: "Ash"},
	{ID: "ballad", Name: "Ballad"},
}
