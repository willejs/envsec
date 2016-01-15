package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"syscall"

	"github.com/kreuzwerker/envsec"
	"github.com/spf13/cobra"
	"github.com/yawn/doubledash"
	"github.com/yawn/envmap"
)

const (
	defaultPrefix = "ENVSEC_"
	version       = "1.0.0"
)

var build string

func main() {

	// global state
	var h envsec.Handler

	os.Args = doubledash.Args

	// flags
	var (
		arn     *string
		prefix  *string
		region  *string
		verbose *bool
	)

	// commands
	root := &cobra.Command{

		Use:   "es",
		Short: "envsec provides encrypted environment variables",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {

			h = envsec.Handler{
				Prefix: *prefix,
			}

			envsec.Verbose = *verbose

		},
	}

	decrypt := &cobra.Command{

		Use:   "dec",
		Short: "decrypt environment variables",
		PreRun: func(cmd *cobra.Command, args []string) {

			method, err := envsec.NewKMSMethod(*region, "")

			if err != nil {
				log.Fatalf("Failed initializing method: %v", err)
			}

			h.Method = method

		},
		Run: func(cmd *cobra.Command, args []string) {

			if len(doubledash.Xtra) < 1 {
				log.Fatal("No command found to execute")
			}

			var (
				env  = h.Decrypt(envmap.Import())
				arg0 = doubledash.Xtra[0]
				argv []string
			)

			if len(doubledash.Xtra) > 1 {
				argv = doubledash.Xtra[1:]
			}

			if err := syscall.Exec(arg0, argv, env); err != nil {
				log.Fatalf("Failed to execute %q", args)
			}

		},
	}

	encrypt := &cobra.Command{

		Use:   "enc",
		Short: "encrypt environment variables",
		PreRun: func(cmd *cobra.Command, args []string) {

			matcher := regexp.MustCompile(`arn\:aws\:kms\:([a-z0-9-]+):\d+\:key\/([a-f0-9\-]+)`)
			matches := matcher.FindAllStringSubmatch(*arn, -1)

			if len(matches) > 0 {

				method, err := envsec.NewKMSMethod(matches[0][1], matches[0][2])

				if err != nil {
					log.Fatalf("Failed initializing method: %v", err)
				}

				h.Method = method

			} else {
				log.Fatalf("Invalid ARN format %q", *arn)
			}

		}, Run: func(cmd *cobra.Command, args []string) {

			for _, e := range h.Encrypt(envmap.ToMap(args)) {
				fmt.Println(e)
			}

		},
	}

	version := &cobra.Command{
		Use:   "version",
		Short: "Print the version information of es",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Envsec %s (%s)\n", version, build)
		},
	}

	root.AddCommand(decrypt)
	root.AddCommand(encrypt)
	root.AddCommand(version)

	// flag parsing

	arn = encrypt.Flags().StringP("arn", "a", "", "ARN of the the AWS KMS key")
	prefix = root.PersistentFlags().StringP("prefix", "p", defaultPrefix, "Prefix distinguishing secure variables")
	region = decrypt.Flags().StringP("region", "r", "eu-west-1", "Default region")
	verbose = root.PersistentFlags().BoolP("verbose", "v", false, "Verbose logging")

	if err := root.Execute(); err != nil {
		log.Fatalf("Failed to start the application: %v", err)
	}

}
