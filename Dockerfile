#
# Copyright 2022 Parallel Wireless
# Copyright 2022 Samsung Electronics Co., Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#
# This is a Dockerfile for code verification only.
# Not to be used for anything else.
#

FROM golang:1.14-stretch as builder

WORKDIR /stslgo
COPY . .

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN go mod download 
RUN go mod tidy

FROM golang:1.14-stretch

WORKDIR /root/

COPY --from=builder /stslgo/ . 
RUN chmod 777 ci/install_influx.sh && \
    ci/install_influx.sh && \
    ci/ci_test.sh
