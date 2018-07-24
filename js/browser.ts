/*
 * Copyright (C) 2018 Red Hat, Inc.
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

declare var window: any;
import apiLib = require('./api');
window.api = apiLib
window.Metadata = apiLib.Metadata
window.NE = apiLib.NE
window.GT = apiLib.GT
window.LT = apiLib.LT
window.GTE = apiLib.GTE
window.LTE = apiLib.LTE
window.IPV4RANGE = apiLib.IPV4RANGE
window.REGEX = apiLib.REGEX
window.WITHIN = apiLib.WITHIN
window.WITHOUT = apiLib.WITHOUT
window.INSIDE = apiLib.INSIDE
window.OUTSIDE = apiLib.OUTSIDE
window.BETWEEN = apiLib.BETWEEN
