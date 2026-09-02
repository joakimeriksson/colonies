package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/colonyos/colonies/internal/crypto"
	"github.com/colonyos/colonies/pkg/client"
	"github.com/colonyos/colonies/pkg/core"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var UsageFrom string
var UsageTo string
var UsageGroupBy string
var UsageProvider string
var UsageInitiator string
var UsageExecutor string

func init() {
	usageCmd.AddCommand(usageLsCmd)
	usageCmd.AddCommand(usageRecordsCmd)
	usageCmd.AddCommand(usageBalanceCmd)
	rootCmd.AddCommand(usageCmd)

	usageCmd.PersistentFlags().StringVarP(&ServerHost, "host", "", DefaultServerHost, "Server host")
	usageCmd.PersistentFlags().IntVarP(&ServerPort, "port", "", -1, "Server HTTP port")
	usageCmd.PersistentFlags().StringVarP(&UsageFrom, "from", "", "30d", "Start of window: a duration back from now (e.g. 24h, 7d) or an RFC3339 time")
	usageCmd.PersistentFlags().StringVarP(&UsageTo, "to", "", "", "End of window as RFC3339 time, default now")
	usageCmd.PersistentFlags().BoolVarP(&JSON, "json", "", false, "Print JSON instead of tables")

	usageLsCmd.Flags().StringVarP(&UsageGroupBy, "group-by", "", core.USAGE_GROUP_INITIATOR, "Group by initiator, provider, executor, executortype or label")
	usageLsCmd.Flags().StringVarP(&UsageProvider, "provider", "", "", "Only usage on executors owned by this user, or \"me\"")
	usageLsCmd.Flags().StringVarP(&UsageInitiator, "initiator", "", "", "Only usage submitted by this user, or \"me\"")

	usageRecordsCmd.Flags().StringVarP(&UsageProvider, "provider", "", "", "Only usage on executors owned by this user, or \"me\"")
	usageRecordsCmd.Flags().StringVarP(&UsageInitiator, "initiator", "", "", "Only usage submitted by this user, or \"me\"")
	usageRecordsCmd.Flags().StringVarP(&UsageExecutor, "executor", "", "", "Only usage on this executor")
	usageRecordsCmd.Flags().IntVarP(&Count, "count", "", 100, "Max number of records")
}

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show who ran what on whose executors",
	Long:  "Show who ran what on whose executors",
}

// parseUsageWindow turns the --from and --to flags into unix seconds.
func parseUsageWindow() (int64, int64) {
	to := time.Now()
	if UsageTo != "" {
		t, err := time.Parse(time.RFC3339, UsageTo)
		CheckError(err)
		to = t
	}

	from := to.Add(-30 * 24 * time.Hour)
	if UsageFrom != "" {
		if strings.HasSuffix(UsageFrom, "d") {
			days, err := strconv.Atoi(strings.TrimSuffix(UsageFrom, "d"))
			CheckError(err)
			from = to.Add(-time.Duration(days) * 24 * time.Hour)
		} else if d, err := time.ParseDuration(UsageFrom); err == nil {
			from = to.Add(-d)
		} else {
			t, err := time.Parse(time.RFC3339, UsageFrom)
			CheckError(err)
			from = t
		}
	}

	return from.Unix(), to.Unix()
}

// resolveUserID maps "", "me" or a user name to a user id.
func resolveUserID(client *client.ColoniesClient, name string) string {
	if name == "" {
		return ""
	}
	if name == "me" {
		id, err := crypto.GenerateID(PrvKey)
		CheckError(err)
		return id
	}
	user, err := client.GetUser(ColonyName, name, PrvKey)
	CheckError(err)
	if user == nil {
		CheckError(errors.New("User <" + name + "> not found in colony <" + ColonyName + ">"))
	}
	return user.ID
}

var usageLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "Summarize usage over a time window",
	Long:  "Summarize reserved core hours, GPU hours and memory over a time window, grouped by initiator, provider, executor, executor type or label",
	Run: func(cmd *cobra.Command, args []string) {
		client := setup()

		from, to := parseUsageWindow()
		providerID := resolveUserID(client, UsageProvider)
		initiatorID := resolveUserID(client, UsageInitiator)

		summaries, err := client.GetUsageSummary(ColonyName, from, to, UsageGroupBy, initiatorID, providerID, PrvKey)
		CheckError(err)

		if JSON {
			jsonString, err := core.ConvertUsageSummaryArrayToJSON(summaries)
			CheckError(err)
			fmt.Println(jsonString)
			os.Exit(0)
		}

		if len(summaries) == 0 {
			log.WithFields(log.Fields{"ColonyName": ColonyName}).Info("No usage found in window")
			os.Exit(0)
		}

		printUsageSummaryTable(summaries, UsageGroupBy)
	},
}

var usageRecordsCmd = &cobra.Command{
	Use:   "records",
	Short: "List individual usage records",
	Long:  "List one record per process attempt: who ran it, on which executor, owned by whom, for how long",
	Run: func(cmd *cobra.Command, args []string) {
		client := setup()

		from, to := parseUsageWindow()
		providerID := resolveUserID(client, UsageProvider)
		initiatorID := resolveUserID(client, UsageInitiator)

		records, err := client.GetUsageRecords(ColonyName, from, to, initiatorID, providerID, UsageExecutor, Count, PrvKey)
		CheckError(err)

		if JSON {
			jsonString, err := core.ConvertUsageRecordArrayToJSON(records)
			CheckError(err)
			fmt.Println(jsonString)
			os.Exit(0)
		}

		if len(records) == 0 {
			log.WithFields(log.Fields{"ColonyName": ColonyName}).Info("No usage records found in window")
			os.Exit(0)
		}

		printUsageRecordsTable(records)
	},
}

var usageBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show provided versus consumed usage per colony member",
	Long:  "Show, per colony member, core hours consumed on others' executors, provided to others on their own executors, and self-use",
	Run: func(cmd *cobra.Command, args []string) {
		client := setup()

		from, to := parseUsageWindow()

		balances, err := client.GetUsageBalance(ColonyName, from, to, PrvKey)
		CheckError(err)

		if JSON {
			jsonString, err := core.ConvertUsageBalanceArrayToJSON(balances)
			CheckError(err)
			fmt.Println(jsonString)
			os.Exit(0)
		}

		if len(balances) == 0 {
			log.WithFields(log.Fields{"ColonyName": ColonyName}).Info("No usage found in window")
			os.Exit(0)
		}

		printUsageBalanceTable(balances)
	},
}
