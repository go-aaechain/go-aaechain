// Copyright 2015 The go-ethereum Authors
// This file is part of the go-aaeereum library.
//
// The go-aaeereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-aaeereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-aaeereum library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/aaechain/go-aaechain/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("aae/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("aae/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("aae/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("aae/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("aae/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("aae/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("aae/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("aae/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("aae/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("aae/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("aae/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("aae/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("aae/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("aae/downloader/states/drop", nil)
)
