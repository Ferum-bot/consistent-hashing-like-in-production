package metrics

import "github.com/prometheus/client_golang/prometheus"

type Migration struct {
	active       prometheus.Gauge
	copied       prometheus.Counter
	gateWaiting  prometheus.Gauge
	gateRejected prometheus.Counter
}

func NewMigration(reg prometheus.Registerer) *Migration {
	m := &Migration{
		active: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "router_migration_active",
			Help: "1 while a new topology is prepared but not committed, writes to moving users wait.",
		}),
		copied: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "router_migration_copied_users_total",
			Help: "Users copied to their new nodes by migrations started on this router.",
		}),
		gateWaiting: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "router_gate_waiting",
			Help: "Writes waiting for a migration to finish.",
		}),
		gateRejected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "router_gate_rejected_total",
			Help: "Writes rejected because a migration took longer than the gate timeout.",
		}),
	}

	reg.MustRegister(m.active, m.copied, m.gateWaiting, m.gateRejected)

	return m
}

func (m *Migration) ObserveMigrationActive(active bool) {
	value := 0.0

	if active {
		value = 1
	}

	m.active.Set(value)
}

func (m *Migration) ObserveCopied(users int) {
	m.copied.Add(float64(users))
}

func (m *Migration) ObserveGateWaiting(delta int) {
	m.gateWaiting.Add(float64(delta))
}

func (m *Migration) ObserveGateRejected() {
	m.gateRejected.Inc()
}
