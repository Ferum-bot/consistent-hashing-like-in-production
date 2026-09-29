package models

import (
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/sharding"
)

// Стратегии шардирования.
const (
	// StrategyMod — hash(user_id) % N.
	StrategyMod = "mod"
	// StrategyRing — кольцо с виртуальными точками.
	StrategyRing = "ring"
)

// Topology — конфигурация кластера: какие есть ноды и какой стратегией по ним раскладываются пользователи.
type Topology struct {
	// Version растёт с каждой новой топологией. Номер присваивает chctl, у всех роутеров он общий.
	Version int64
	// Strategy — имя стратегии шардирования: mod или ring.
	Strategy string
	// VNodes — сколько точек на кольце у ноды с весом 1. Нужно только кольцу.
	VNodes int
	Nodes  []sharding.Node
	// MembershipDelay — через сколько после получения роутер с этим именем применит топологию.
	// Так изображается роутер, который узнаёт об изменениях кластера с опозданием.
	MembershipDelay map[string]time.Duration
	// Fencing — защита по версии: роутер ставит версию топологии на запросы к нодам,
	// а ноды отклоняют запросы со старой версией и без версии.
	Fencing bool
}

// TopologyStats — что получилось после применения топологии.
type TopologyStats struct {
	Version int64
	// KeyShare — какая доля пользователей по стратегии достаётся каждой ноде.
	KeyShare map[string]float64
	// MovedShare — у какой доли пользователей сменилась нода по сравнению с прошлой топологией.
	MovedShare float64
	// RingPoints — сколько точек на кольце. Для стратегий без кольца — число нод.
	RingPoints int
}

// RouterState — что роутер знает о топологии прямо сейчас.
type RouterState struct {
	Topology Topology
	// PendingVersion — версия подготовленной, но ещё не переключённой топологии. 0 — переноса нет.
	PendingVersion int64
	// DeferredError — почему не применилось последнее отложенное изменение отстающего роутера.
	DeferredError string
}

// Route — куда роутер отправит пользователя и почему.
type Route struct {
	UserID int64
	// Key — то, что хешируем: user_id в виде строки.
	Key  string
	Hash uint64
	Node sharding.Node
	// Version и Strategy — топология, по которой выбрана нода.
	Version  int64
	Strategy string
	// Point — точка кольца, на которую попал ключ. Есть только у кольца.
	Point *RingPoint
	// FenceVersion — версия топологии для заголовка запроса к ноде. 0 — защита выключена.
	FenceVersion int64
}

// RingPoint — точка на кольце: чья она, какая по счёту у своей ноды и где стоит.
type RingPoint struct {
	Node  string
	Index int
	Hash  uint64
}
