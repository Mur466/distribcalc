package cfg

import (
	"flag"
)

type Config struct {
	Dbhost      string
	Dbuser      string
	Dbpassword  string
	Dbname      string
	Dbport      int
	HttpPort    int
	DelayForAdd int
	DelayForSub int
	DelayForMul int
	DelayForDiv int
	RowsOnPage  int
	AgentLostTimeout int
}

var Cfg Config

func InitConfig() {

	flag.StringVar(&Cfg.Dbhost, "dbhost", "localhost", "Postgress host")
	flag.StringVar(&Cfg.Dbuser, "dbuser", "postgres", "Postgress user")
	flag.StringVar(&Cfg.Dbpassword, "dbpassword", "postgres", "Postgress password")
	flag.IntVar(&Cfg.Dbport, "dbport", 5432, "Posgress port")
	flag.StringVar(&Cfg.Dbname, "dbname", "distribcalc", "Postgress database name")
	flag.IntVar(&Cfg.HttpPort, "httppport", 8080, "HTTP port to listen to")
	flag.IntVar(&Cfg.AgentLostTimeout, "agenttimeout", 60, "Timeout before agent considered lost (seconds)")

	flag.Parse()

	Cfg.DelayForAdd = 10
	Cfg.DelayForSub = 12
	Cfg.DelayForMul = 15
	Cfg.DelayForDiv = 20

	Cfg.RowsOnPage = 10
}

func RecalcAgentTimeout() {
	// таймаут для агента вдвое дольше самой долгой операции
	// вообще в принципе они особо не связаны
	// просто имеем значение тоже же порядка, что пользователь задал в интерфейсе настроек
	Cfg.AgentLostTimeout = 2 * Max(
		Cfg.DelayForAdd,
		Cfg.DelayForSub,
		Cfg.DelayForMul,
		Cfg.DelayForDiv,
	)
}

func Max(vals ...int) int {
	if len(vals) == 0 {
		panic("Give at least one argument")
	}
	m := vals[0]
	for v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}

