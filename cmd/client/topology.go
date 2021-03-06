/*
 * Copyright (C) 2016 Red Hat, Inc.
 *
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/skydive-project/skydive/analyzer"
	"github.com/skydive-project/skydive/common"
	"github.com/skydive-project/skydive/config"
	"github.com/skydive-project/skydive/topology/graph"
	"github.com/skydive-project/skydive/websocket"
	"github.com/spf13/cobra"
)

var (
	gremlinQuery string
	outputFormat string
	filename     string
)

// TopologyCmd skydive topology root command
var TopologyCmd = &cobra.Command{
	Use:          "topology",
	Short:        "Request on topology [deprecated: use 'client query' instead]",
	Long:         "Request on topology [deprecated: use 'client query' instead]",
	SilenceUsage: false,
}

// TopologyRequest skydive topology query command
var TopologyRequest = &cobra.Command{
	Use:   "query",
	Short: "query topology",
	Long:  "query topology",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "The 'client topology query' command is deprecated. Please use 'client query' instead")
		QueryCmd.Run(cmd, []string{gremlinQuery})
	},
}

// TopologyImport skydive topology import command
var TopologyImport = &cobra.Command{
	Use:   "import",
	Short: "import topology",
	Long:  "import topology",
	Run: func(cmd *cobra.Command, args []string) {
		sa, err := config.GetOneAnalyzerServiceAddress()
		if err != nil {
			exitOnError(err)
		}

		url := config.GetURL("ws", sa.Addr, sa.Port, "/ws/publisher")
		headers := http.Header{}
		headers.Add("X-Persistence-Policy", string(analyzer.Persistent))
		client, err := config.NewWSClient(common.UnknownService, url, &AuthenticationOpts, headers)
		if err != nil {
			exitOnError(err)
		}

		if err := client.Connect(); err != nil {
			exitOnError(err)
		}

		go client.Run()
		defer func() {
			client.Flush()
			client.Conn.Stop()
		}()

		content, err := ioutil.ReadFile(filename)
		if err != nil {
			exitOnError(err)
		}

		syncMsg := []*graph.SyncMsg{}
		if err := json.Unmarshal(content, &syncMsg); err != nil {
			exitOnError(err)
		}

		if len(syncMsg) != 1 {
			exitOnError(errors.New("Invalid graph format"))
		}

		for _, node := range syncMsg[0].Nodes {
			msg := websocket.NewStructMessage(graph.Namespace, graph.NodeAddedMsgType, node)
			if err := client.SendMessage(msg); err != nil {
				exitOnError(fmt.Errorf("Failed to send message: %s", err))
			}
		}

		for _, edge := range syncMsg[0].Edges {
			msg := websocket.NewStructMessage(graph.Namespace, graph.EdgeAddedMsgType, edge)
			if err := client.SendMessage(msg); err != nil {
				exitOnError(fmt.Errorf("Failed to send message: %s", err))
			}
		}
	},
}

// TopologyExport skydive topology export command
var TopologyExport = &cobra.Command{
	Use:   "export",
	Short: "export topology",
	Long:  "export topology",
	Run: func(cmd *cobra.Command, args []string) {
		QueryCmd.Run(cmd, []string{"G"})
	},
}

func init() {
	TopologyCmd.AddCommand(TopologyExport)

	TopologyImport.Flags().StringVarP(&filename, "file", "", "graph.json", "Input file")
	TopologyCmd.AddCommand(TopologyImport)

	TopologyRequest.Flags().StringVarP(&gremlinQuery, "gremlin", "", "G", "Gremlin Query")
	TopologyRequest.Flags().StringVarP(&outputFormat, "format", "", "json", "Output format (json, dot or pcap)")
	TopologyCmd.AddCommand(TopologyRequest)
}
