package tsdbctl

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/v3io/v3io-tsdb/pkg/querier"
	"strconv"
	"strings"
	"time"
	"github.com/v3io/v3io-tsdb/pkg/formatter"
)

type queryCommandeer struct {
	cmd            *cobra.Command
	rootCommandeer *RootCommandeer
	name           string
	filter         string
	//to             string
	//from           string
	last      string
	windows   string
	functions string
	step      string
	output    string
}

func newQueryCommandeer(rootCommandeer *RootCommandeer) *queryCommandeer {
	commandeer := &queryCommandeer{
		rootCommandeer: rootCommandeer,
	}

	cmd := &cobra.Command{
		Use:     "query name [flags]",
		Aliases: []string{"get"},
		Short:   "query time series metrics",
		RunE: func(cmd *cobra.Command, args []string) error {

			// if we got positional arguments
			if len(args) > 0 {
				commandeer.name = args[0]
			}

			return commandeer.query()

		},
	}

	//cmd.Flags().StringVarP(&commandeer.to, "to", "t", "", "to time")
	//cmd.Flags().StringVarP(&commandeer.from, "from", "f", "", "from time")
	cmd.Flags().StringVarP(&commandeer.output, "output", "o", "", "output format: text,csv,json")
	cmd.Flags().StringVarP(&commandeer.filter, "filter", "f", "", "v3io query filter e.g. method=='get'")
	cmd.Flags().StringVarP(&commandeer.last, "last", "l", "", "last min/hours/days e.g. 15m")
	cmd.Flags().StringVarP(&commandeer.windows, "windows", "w", "", "comma separated list of overlapping windows")
	cmd.Flags().StringVarP(&commandeer.functions, "aggregators", "a", "",
		"comma separated list of aggregation functions, e.g. count,avg,sum,min,max,stddev,stdvar")
	cmd.Flags().StringVarP(&commandeer.step, "step", "i", "", "interval step for aggregation functions")

	commandeer.cmd = cmd

	return commandeer
}

func (qc *queryCommandeer) query() error {

	if qc.name == "" && qc.filter == "" {
		return errors.New("query must have a metric name or filter string")
	}
	// initialize psrsmd snd adapter
	if err := qc.rootCommandeer.initialize(); err != nil {
		return err
	}

	if err := qc.rootCommandeer.startAdapter(); err != nil {
		return err
	}

	step, err := str2duration(qc.step)
	if err != nil {
		return err
	}

	// TODO: start & end times
	to := time.Now().Unix() * 1000
	from := to - 1000*3600 // default of last hour
	if qc.last != "" {
		last, err := str2duration(qc.last)
		if err != nil {
			return err
		}
		from = to - last
	}

	qc.rootCommandeer.logger.DebugWith("Query", "from", from, "to", to, "name", qc.name,
		"filter", qc.filter, "functions", qc.functions, "step", qc.step)

	qry, err := qc.rootCommandeer.adapter.Querier(nil, from, to)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize Querier")
	}

	var set querier.SeriesSet
	if qc.windows == "" {
		set, err = qry.Select(qc.name, qc.functions, step, qc.filter)
	} else {
		list := strings.Split(qc.windows, ",")
		win := []int{}
		for _, val := range list {
			i, err := strconv.Atoi(val)
			if err != nil {
				return errors.Wrap(err, "not a valid window")
			}
			win = append(win, i)

		}

		set, err = qry.SelectOverlap(qc.name, qc.functions, step, win, qc.filter)
	}

	if err != nil {
		return errors.Wrap(err, "Select Failed")
	}

	f, err := formatter.NewFormatter(qc.output, nil)
	if err != nil {
		return errors.Wrap(err, "failed to start formatter " + qc.output)
	}

	return f.Write(qc.cmd.OutOrStdout(), set)


	//return qc.printSet(set)
}

func (qc *queryCommandeer) printSet(set querier.SeriesSet) error {

	for set.Next() {
		if set.Err() != nil {
			return set.Err()
		}

		series := set.At()
		fmt.Println("Lables:", series.Labels())
		iter := series.Iterator()
		for iter.Next() {

			if iter.Err() != nil {
				return iter.Err()
			}

			t, v := iter.At()
			timeString := time.Unix(t/1000, 0).Format(time.RFC3339)
			fmt.Printf("%s  v=%.2f\n", timeString, v)
		}
		fmt.Println()
	}

	return nil
}

func str2duration(duration string) (int64, error) {

	multiply := 3600 * 1000  // hour by default
	if len(duration) > 0 {
		last := duration[len(duration)-1:]
		if last == "m" || last == "h" || last == "d" {
			duration = duration[0 : len(duration)-1]
			switch last {
			case "m":
				multiply = 60 * 1000
			case "h":
				multiply = 3600 * 1000
			case "d":
				multiply = 24 * 3600 * 1000
			}
		}
	}

	if duration == "" {
		return 0, nil
	}

	i, err := strconv.Atoi(duration)
	if err != nil {
		return 0, errors.Wrap(err, "not a valid duration")
	}

	return int64(i * multiply), nil
}
