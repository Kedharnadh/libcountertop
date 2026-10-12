package schema

import "github.com/ygelfand/libcountertop/pkg/settings/storage"

type Shared struct {
	Alerts    Alerts    `json:"alerts"`
	API       API       `json:"api"`
	Apps      Apps      `json:"apps"`
	Bluetooth Bluetooth `json:"bluetooth"`
	Calendar  Calendar  `json:"calendar"`
	Call      Call      `json:"call"`
	Camera    Camera    `json:"camera"`
	Cast      Cast      `json:"cast"`
	Clock     Clock     `json:"clock"`
	Feedback  Feedback  `json:"feedback"`
	Home      Home      `json:"home"`
	Idle      Idle      `json:"idle"`
	Media     Media     `json:"media"`
	Network   Network   `json:"network"`
	Poster    Poster    `json:"poster"`
	Radio     Radio     `json:"radio"`
	RTSP      RTSP      `json:"rtsp"`
	Screen    Screen    `json:"screen"`
	Sendspin  Sendspin  `json:"sendspin"`
	Time      Time      `json:"time"`
	Todo      Todo      `json:"todo"`
	Update    Update    `json:"update"`
	Visual    Visual    `json:"visual"`
	Voice     Voice     `json:"voice"`
	Wake      Wake      `json:"wake"`
	Weather   Weather   `json:"weather"`
}

func DefaultShared() Shared {
	return Shared{
		Alerts:    DefaultAlerts(),
		API:       DefaultAPI(),
		Apps:      DefaultApps(),
		Bluetooth: DefaultBluetooth(),
		Calendar:  DefaultCalendar(),
		Call:      DefaultCall(),
		Camera:    DefaultCamera(),
		Cast:      DefaultCast(),
		Clock:     DefaultClock(),
		Feedback:  DefaultFeedback(),
		Home:      DefaultHome(),
		Idle:      DefaultIdle(),
		Media:     DefaultMedia(),
		Network:   DefaultNetwork(),
		Poster:    DefaultPoster(),
		Radio:     DefaultRadio(),
		RTSP:      DefaultRTSP(),
		Screen:    DefaultScreen(),
		Sendspin:  DefaultSendspin(),
		Todo:      DefaultTodo(),
		Update:    DefaultUpdate(),
		Visual:    DefaultVisualSettings(),
		Voice:     DefaultVoice(),
		Wake:      DefaultWake(),
		Weather:   DefaultWeather(),
	}
}

func (s Shared) Clone() Shared {
	s.Wake = s.Wake.Clone()
	s.Cast = s.Cast.Clone()
	s.Camera = s.Camera.Clone()
	s.Home = s.Home.Clone()
	s.Apps = s.Apps.Clone()
	s.Radio = s.Radio.Clone()
	return s
}

type Sections[C any] struct {
	Alerts    storage.Binding[C, Alerts]
	API       storage.Binding[C, API]
	Apps      storage.Binding[C, Apps]
	Bluetooth storage.Binding[C, Bluetooth]
	Calendar  storage.Binding[C, Calendar]
	Call      storage.Binding[C, Call]
	Camera    storage.Binding[C, Camera]
	Cast      storage.Binding[C, Cast]
	Clock     storage.Binding[C, Clock]
	Feedback  storage.Binding[C, Feedback]
	Home      storage.Binding[C, Home]
	Idle      storage.Binding[C, Idle]
	Media     storage.Binding[C, Media]
	Network   storage.Binding[C, Network]
	Poster    storage.Binding[C, Poster]
	Radio     storage.Binding[C, Radio]
	RTSP      storage.Binding[C, RTSP]
	Screen    storage.Binding[C, Screen]
	Sendspin  storage.Binding[C, Sendspin]
	Time      storage.Binding[C, Time]
	Todo      storage.Binding[C, Todo]
	Update    storage.Binding[C, Update]
	Visual    storage.Binding[C, Visual]
	Voice     storage.Binding[C, Voice]
	Wake      storage.Binding[C, Wake]
	Weather   storage.Binding[C, Weather]
}

func NewSections[C any](store func() *storage.Store[C], shared func(*C) *Shared) Sections[C] {
	return Sections[C]{
		Alerts:    bind(store, shared, func(s *Shared) *Alerts { return &s.Alerts }),
		API:       bind(store, shared, func(s *Shared) *API { return &s.API }),
		Apps:      bind(store, shared, func(s *Shared) *Apps { return &s.Apps }),
		Bluetooth: bind(store, shared, func(s *Shared) *Bluetooth { return &s.Bluetooth }),
		Calendar:  bind(store, shared, func(s *Shared) *Calendar { return &s.Calendar }),
		Call:      bind(store, shared, func(s *Shared) *Call { return &s.Call }),
		Camera:    bind(store, shared, func(s *Shared) *Camera { return &s.Camera }),
		Cast:      bind(store, shared, func(s *Shared) *Cast { return &s.Cast }),
		Clock:     bind(store, shared, func(s *Shared) *Clock { return &s.Clock }),
		Feedback:  bind(store, shared, func(s *Shared) *Feedback { return &s.Feedback }),
		Home:      bind(store, shared, func(s *Shared) *Home { return &s.Home }),
		Idle:      bind(store, shared, func(s *Shared) *Idle { return &s.Idle }),
		Media:     bind(store, shared, func(s *Shared) *Media { return &s.Media }),
		Network:   bind(store, shared, func(s *Shared) *Network { return &s.Network }),
		Poster:    bind(store, shared, func(s *Shared) *Poster { return &s.Poster }),
		Radio:     bind(store, shared, func(s *Shared) *Radio { return &s.Radio }),
		RTSP:      bind(store, shared, func(s *Shared) *RTSP { return &s.RTSP }),
		Screen:    bind(store, shared, func(s *Shared) *Screen { return &s.Screen }),
		Sendspin:  bind(store, shared, func(s *Shared) *Sendspin { return &s.Sendspin }),
		Time:      bind(store, shared, func(s *Shared) *Time { return &s.Time }),
		Todo:      bind(store, shared, func(s *Shared) *Todo { return &s.Todo }),
		Update:    bind(store, shared, func(s *Shared) *Update { return &s.Update }),
		Visual:    bind(store, shared, func(s *Shared) *Visual { return &s.Visual }),
		Voice:     bind(store, shared, func(s *Shared) *Voice { return &s.Voice }),
		Wake:      bind(store, shared, func(s *Shared) *Wake { return &s.Wake }),
		Weather:   bind(store, shared, func(s *Shared) *Weather { return &s.Weather }),
	}
}

func bind[C, T any](
	store func() *storage.Store[C],
	shared func(*C) *Shared,
	field func(*Shared) *T,
) storage.Binding[C, T] {
	return storage.Bind(store, func(c *C) *T { return field(shared(c)) })
}

func (s Sections[C]) Writer() SharedWriter[C] { return SharedWriter[C]{s: s} }

type SharedWriter[C any] struct{ s Sections[C] }

func (w SharedWriter[C]) API() APIWriter             { return NewAPIWriter(w.s.API) }
func (w SharedWriter[C]) Bluetooth() BluetoothWriter { return NewBluetoothWriter(w.s.Bluetooth) }
func (w SharedWriter[C]) Calendar() CalendarWriter   { return NewCalendarWriter(w.s.Calendar) }
func (w SharedWriter[C]) Call() CallWriter           { return NewCallWriter(w.s.Call) }
func (w SharedWriter[C]) Camera() CameraWriter       { return NewCameraWriter(w.s.Camera) }
func (w SharedWriter[C]) Cast() CastWriter           { return NewCastWriter(w.s.Cast) }
func (w SharedWriter[C]) Clock() ClockWriter         { return NewClockWriter(w.s.Clock) }
func (w SharedWriter[C]) Feedback() FeedbackWriter   { return NewFeedbackWriter(w.s.Feedback) }
func (w SharedWriter[C]) Home() HomeWriter           { return NewHomeWriter(w.s.Home) }
func (w SharedWriter[C]) Idle() IdleWriter           { return NewIdleWriter(w.s.Idle) }
func (w SharedWriter[C]) Media() MediaWriter         { return NewMediaWriter(w.s.Media) }
func (w SharedWriter[C]) Network() NetworkWriter     { return NewNetworkWriter(w.s.Network) }
func (w SharedWriter[C]) Poster() PosterWriter       { return NewPosterWriter(w.s.Poster) }
func (w SharedWriter[C]) Radio() RadioWriter         { return NewRadioWriter(w.s.Radio) }
func (w SharedWriter[C]) RTSP() RTSPWriter           { return NewRTSPWriter(w.s.RTSP) }
func (w SharedWriter[C]) Screen() ScreenWriter       { return NewScreenWriter(w.s.Screen) }
func (w SharedWriter[C]) Sendspin() SendspinWriter   { return NewSendspinWriter(w.s.Sendspin) }
func (w SharedWriter[C]) Time() TimeWriter           { return NewTimeWriter(w.s.Time) }
func (w SharedWriter[C]) Todo() TodoWriter           { return NewTodoWriter(w.s.Todo) }
func (w SharedWriter[C]) Update() UpdateWriter       { return NewUpdateWriter(w.s.Update) }
func (w SharedWriter[C]) Visual() VisualWriter       { return NewVisualWriter(w.s.Visual) }
func (w SharedWriter[C]) Wake(slot int) WakeWriter   { return NewWakeWriter(w.s.Wake, slot) }
func (w SharedWriter[C]) Stop() StopWriter           { return NewStopWriter(w.s.Wake) }
func (w SharedWriter[C]) Weather() WeatherWriter     { return NewWeatherWriter(w.s.Weather) }
