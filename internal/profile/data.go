package profile

type SiteData struct {
	Person       Person         `json:"person"`
	Hero         Hero           `json:"hero"`
	Contacts     []Contact      `json:"contacts"`
	Navigation   []NavItem      `json:"navigation"`
	Metrics      []Metric       `json:"metrics"`
	Skills       []SkillGroup   `json:"skills"`
	Experiences  []Experience   `json:"experiences"`
	Projects     []Project      `json:"projects"`
	Research     []ResearchItem `json:"research"`
	Coding       CodingProfile  `json:"coding"`
	Reading      []ReadingItem  `json:"reading"`
	BlogIdeas    []BlogItem     `json:"blogIdeas"`
	TechRadar    []RadarItem    `json:"techRadar"`
	Activity     []ActivityDay  `json:"activity"`
	ServiceStack []StackLayer   `json:"serviceStack"`
}

type Person struct {
	Name       string   `json:"name"`
	English    string   `json:"english"`
	Title      string   `json:"title"`
	School     string   `json:"school"`
	Major      string   `json:"major"`
	Location   string   `json:"location"`
	Intentions []string `json:"intentions"`
	Summary    string   `json:"summary"`
}

type Hero struct {
	Eyebrow     string   `json:"eyebrow"`
	Headline    string   `json:"headline"`
	Subheadline string   `json:"subheadline"`
	Focus       []string `json:"focus"`
}

type Contact struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Value string `json:"value"`
	Href  string `json:"href"`
}

type NavItem struct {
	Label string `json:"label"`
	To    string `json:"to"`
}

type Metric struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Suffix string `json:"suffix"`
	Note   string `json:"note"`
}

type SkillGroup struct {
	Name        string   `json:"name"`
	Level       int      `json:"level"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
}

type Experience struct {
	Company string   `json:"company"`
	Role    string   `json:"role"`
	Period  string   `json:"period"`
	Place   string   `json:"place"`
	Stack   []string `json:"stack"`
	Points  []string `json:"points"`
}

type Project struct {
	Name       string   `json:"name"`
	Role       string   `json:"role"`
	Period     string   `json:"period"`
	Link       string   `json:"link"`
	Stack      []string `json:"stack"`
	Highlights []string `json:"highlights"`
	Impact     string   `json:"impact"`
}

type ResearchItem struct {
	Title  string   `json:"title"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
	Detail string   `json:"detail"`
}

type CodingProfile struct {
	Headline string         `json:"headline"`
	Metrics  []CodingMetric `json:"metrics"`
	Tracks   []CodingTrack  `json:"tracks"`
}

type CodingMetric struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Unit  string `json:"unit"`
	Trend string `json:"trend"`
}

type CodingTrack struct {
	Name     string `json:"name"`
	Progress int    `json:"progress"`
	Note     string `json:"note"`
}

type ReadingItem struct {
	Topic    string   `json:"topic"`
	Cadence  string   `json:"cadence"`
	Keywords []string `json:"keywords"`
	Summary  string   `json:"summary"`
}

type BlogItem struct {
	Title string   `json:"title"`
	Type  string   `json:"type"`
	Tags  []string `json:"tags"`
	Brief string   `json:"brief"`
}

type RadarItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type ActivityDay struct {
	Day   string `json:"day"`
	Score int    `json:"score"`
	Kind  string `json:"kind"`
}

type StackLayer struct {
	Name  string   `json:"name"`
	Items []string `json:"items"`
}

type Signal struct {
	Level     string `json:"level"`
	Label     string `json:"label"`
	Detail    string `json:"detail"`
	Progress  int    `json:"progress"`
	Timestamp string `json:"timestamp"`
}

func Data() SiteData {
	return SiteData{
		Person: Person{
			Name:     "张孟庆",
			English:  "Mengqing Zhang",
			Title:    "2027届硕士 | 后端开发 / 数据处理 / AI应用开发",
			School:   "中国农业大学 信息与电气工程学院",
			Major:    "计算机科学与技术",
			Location: "北京 海淀区",
			Intentions: []string{
				"软件开发",
				"后端开发",
				"数据处理",
				"AI应用开发",
				"企业信息化建设",
			},
			Summary: "中国农业大学计算机科学与技术学术型硕士，具备央企算法实习和国家科研项目经历。熟悉 Python、Go、Java、SQL，具备 Spring Boot、FastAPI、React/TypeScript 平台开发实践，能够完成数据处理、模块开发、接口对接和结果可视化。",
		},
		Hero: Hero{
			Eyebrow:     "Computer Science Graduate Student",
			Headline:    "深度学习视觉方向，主要使用遥感数据。",
			Subheadline: "",
			Focus: []string{
				"Go API Service",
				"Vue + Element Plus",
				"PyTorch CV",
			},
		},
		Contacts: []Contact{
			{Type: "phone", Label: "电话", Value: "18513309077", Href: "tel:18513309077"},
			{Type: "mail", Label: "邮箱", Value: "zhangmengqing@cau.edu.cn", Href: "mailto:zhangmengqing@cau.edu.cn"},
			{Type: "pin", Label: "城市", Value: "北京 海淀区"},
			{Type: "link", Label: "遥感数据平台", Value: "iqboshi.github.io/platform", Href: "https://iqboshi.github.io/platform/"},
		},
		Navigation: []NavItem{
			{Label: "能力", To: "#skills"},
			{Label: "项目", To: "#projects"},
			{Label: "科研", To: "#research"},
			{Label: "编码", To: "#coding"},
			{Label: "阅读", To: "#reading"},
		},
		Metrics: []Metric{
			{Label: "硕士阶段", Value: "2027", Suffix: "届", Note: "中国农业大学 985"},
			{Label: "科研/工程项目", Value: "3", Suffix: "+", Note: "遥感、病虫害、智慧施肥"},
			{Label: "央企算法实习", Value: "6", Suffix: "月", Note: "强化学习与 LLM 可视化"},
			{Label: "在投一作论文", Value: "2", Suffix: "篇", Note: "中科院一区方向"},
		},
		Skills: []SkillGroup{
			{
				Name:        "编程与基础",
				Level:       90,
				Description: "熟悉 Python、Go、Java、SQL，具备脚本开发、数据处理、算法模块封装和工程调试经验。",
				Keywords:    []string{"Python", "Go", "Java", "SQL", "Data Structures"},
			},
			{
				Name:        "后端与平台",
				Level:       84,
				Description: "了解 Spring Boot REST 接口开发流程，熟悉 FastAPI 接口开发和 MySQL/Redis 常见使用方式。",
				Keywords:    []string{"Spring Boot", "FastAPI", "MySQL", "Redis", "REST API"},
			},
			{
				Name:        "AI/CV能力",
				Level:       88,
				Description: "熟悉 PyTorch 模型训练流程，掌握目标检测、语义分割、实例分割等视觉任务。",
				Keywords:    []string{"PyTorch", "目标检测", "语义分割", "实例分割", "遥感影像"},
			},
			{
				Name:        "工程协作",
				Level:       82,
				Description: "熟悉 Git/GitHub 版本管理、Linux 远程开发、AutoDL 训练环境和实验管理。",
				Keywords:    []string{"Git", "GitHub", "Linux", "AutoDL", "Issue Tracking"},
			},
			{
				Name:        "大模型应用",
				Level:       76,
				Description: "了解 LLM 应用开发、RAG 基本原理和 Agent 调试流程，具备推理过程可视化实践。",
				Keywords:    []string{"LLM", "RAG", "Agent", "可解释性", "推理可视化"},
			},
		},
		Experiences: []Experience{
			{
				Company: "中国船舶集团有限公司",
				Role:    "算法工程师实习",
				Period:  "2024.08 - 2025.02",
				Place:   "北京",
				Stack:   []string{"Python", "强化学习", "LLM", "可视化", "模型可解释性分析"},
				Points: []string{
					"围绕强化学习与 LLM 模型调试场景，负责算法结果可视化模块开发，完成状态解析、路径渲染和回放逻辑。",
					"将智能体决策路径、推理中间步骤和注意力信息进行结构化展示，帮助定位模型异常并解释决策过程。",
					"参与模块联调、实验记录和演示材料整理，配合算法团队完成结果分析和跨模块展示。",
				},
			},
		},
		Projects: []Project{
			{
				Name:   "遥感数据处理与流程管理平台",
				Role:   "全栈开发",
				Period: "在线作品",
				Link:   "https://iqboshi.github.io/platform/",
				Stack:  []string{"React", "TypeScript", "Ant Design", "FastAPI", "Python"},
				Highlights: []string{
					"搭建遥感数据管理系统，包含数据集版本、地图预览、处理流程编辑和结果管理。",
					"使用 React、TypeScript、Ant Design 开发前端页面，FastAPI 提供接口、权限、上传和流程校验。",
				},
				Impact: "在线预览可直接查看交互，本地版可运行 FastAPI 和 SQLite，能看到真实接口和数据表设计。",
			},
			{
				Name:   "水稻病虫害分级与按需喷药系统",
				Role:   "算法开发",
				Period: "2024.12 - 至今",
				Stack:  []string{"Python", "PyTorch", "计算机视觉", "遥感可视化", "病虫害分级"},
				Highlights: []string{
					"面向复杂稻田场景，负责病虫害监测与分级算法实验，完成数据整理、模型训练和效果对比。",
					"参与病虫害空间分布可视化平台开发，把模型识别结果整理成喷药作业可参考的信息。",
				},
				Impact: "把视觉模型结果接到真实农业任务里，关注结果是否能被使用者看懂和采用。",
			},
			{
				Name:   "农田智慧施肥与冬小麦分蘖密度无人机遥感监测",
				Role:   "算法开发",
				Period: "2022.06 - 至今",
				Stack:  []string{"Python", "无人机遥感", "PROSAIL 反演", "迁移学习", "模型验证"},
				Highlights: []string{
					"参与无人机遥感影像和田间采样数据处理，围绕 PROSAIL 反演和迁移学习开展算法优化实验。",
					"负责算法与无人机系统的数据流程对接和验证，检查算法能否用于真实农田任务。",
				},
				Impact: "长期处理真实农田数据，关注算法结果能否落到具体任务里。",
			},
		},
		Research: []ResearchItem{
			{
				Title:  "农业工程学报论文",
				Status: "论文 1 篇",
				Tags:   []string{"农业影像", "遥感应用", "模型验证"},
				Detail: "围绕农业影像与智能监测整理实验结果，并把方法、数据和结论写成论文材料。",
			},
			{
				Title:  "中科院一区方向一作论文",
				Status: "2 篇在投",
				Tags:   []string{"一作", "计算机视觉", "智慧农业"},
				Detail: "围绕复杂农业场景中的视觉识别、区域分析和作业建议展开。",
			},
		},
		Coding: CodingProfile{
			Headline: "算法训练与工程节奏",
			Metrics: []CodingMetric{
				{Label: "LeetCode 题量", Value: "--", Unit: "题", Trend: "预留同步位置"},
				{Label: "连续打卡", Value: "--", Unit: "天", Trend: "可接入打卡记录"},
				{Label: "工程提交节奏", Value: "周更", Unit: "", Trend: "项目与论文并行迭代"},
				{Label: "重点题型", Value: "DP / 图 / 数组", Unit: "", Trend: "保持基础题手感"},
			},
			Tracks: []CodingTrack{
				{Name: "数据结构基础", Progress: 88, Note: "数组、链表、哈希、栈队列"},
				{Name: "动态规划", Progress: 70, Note: "状态设计与转移压缩"},
				{Name: "图搜索", Progress: 76, Note: "BFS、DFS、最短路"},
				{Name: "SQL 与后端题", Progress: 82, Note: "查询优化与接口设计"},
			},
		},
		Reading: []ReadingItem{
			{
				Topic:    "无人机遥感与作物表型",
				Cadence:  "长期跟踪",
				Keywords: []string{"UAV", "PROSAIL", "迁移学习", "模型验证"},
				Summary:  "关注无人机影像、田间采样数据和作物生长指标如何进入模型，以及结果怎么验证。",
			},
			{
				Topic:    "计算机视觉农业应用",
				Cadence:  "项目驱动",
				Keywords: []string{"检测", "分割", "病虫害分级", "空间可视化"},
				Summary:  "关注复杂稻田场景里的目标识别、区域分割和作业建议表达，记录数据处理和模型评估方法。",
			},
			{
				Topic:    "LLM Agent 与可解释性",
				Cadence:  "实习延展",
				Keywords: []string{"RAG", "Agent", "Trajectory", "Reasoning Trace"},
				Summary:  "关注推理过程结构化、路径回放、注意力信息展示和异常定位。",
			},
		},
		BlogIdeas: []BlogItem{
			{
				Title: "Go 后端如何承载个人主页数据 API",
				Type:  "Engineering Note",
				Tags:  []string{"Go", "net/http", "SSE"},
				Brief: "用轻量服务提供结构化站点数据、实时状态流和前端静态托管。",
			},
			{
				Title: "遥感数据平台的模块拆解",
				Type:  "Project Review",
				Tags:  []string{"React", "FastAPI", "Workflow"},
				Brief: "记录数据集、地图页面、模型结果和流程校验几个部分的设计。",
			},
			{
				Title: "从模型输出到喷药作业建议",
				Type:  "Research Log",
				Tags:  []string{"PyTorch", "CV", "Smart Agriculture"},
				Brief: "记录病虫害分级、空间分布表达和农业作业建议之间的连接方式。",
			},
		},
		TechRadar: []RadarItem{
			{Name: "Go/API", Value: 86},
			{Name: "Python/Data", Value: 92},
			{Name: "CV/PyTorch", Value: 88},
			{Name: "Frontend", Value: 82},
			{Name: "Research", Value: 84},
			{Name: "LLM/RAG", Value: 74},
		},
		Activity: []ActivityDay{
			{Day: "Mon", Score: 2, Kind: "code"},
			{Day: "Tue", Score: 4, Kind: "paper"},
			{Day: "Wed", Score: 3, Kind: "experiment"},
			{Day: "Thu", Score: 5, Kind: "code"},
			{Day: "Fri", Score: 4, Kind: "project"},
			{Day: "Sat", Score: 2, Kind: "reading"},
			{Day: "Sun", Score: 3, Kind: "review"},
			{Day: "Mon", Score: 5, Kind: "experiment"},
			{Day: "Tue", Score: 4, Kind: "code"},
			{Day: "Wed", Score: 2, Kind: "paper"},
			{Day: "Thu", Score: 3, Kind: "reading"},
			{Day: "Fri", Score: 5, Kind: "project"},
			{Day: "Sat", Score: 4, Kind: "code"},
			{Day: "Sun", Score: 2, Kind: "rest"},
			{Day: "Mon", Score: 3, Kind: "paper"},
			{Day: "Tue", Score: 5, Kind: "experiment"},
			{Day: "Wed", Score: 4, Kind: "code"},
			{Day: "Thu", Score: 5, Kind: "project"},
		},
		ServiceStack: []StackLayer{
			{Name: "Backend", Items: []string{"Go net/http", "JSON API", "Server-Sent Events", "Static SPA hosting"}},
			{Name: "Frontend", Items: []string{"Vue 3", "Element Plus", "Blog feed layout", "Topic filters"}},
			{Name: "AI/Data", Items: []string{"PyTorch", "Vision data", "Task validation", "Reasoning visualization"}},
		},
	}
}
