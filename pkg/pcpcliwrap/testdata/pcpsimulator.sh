#!/usr/bin/env bash
set -euo pipefail

TESTOUT="$0.testout"
po() {
  printf "%s\n" "$@"
}
pe() {
  printf "%s\n" "$@" >&2
}
pt() {
  printf "%s\n" "$@" >>"$TESTOUT"
}

# Init testout
printf "" >"$TESTOUT"

EXPECTED_ARGV=(
  --protocol 6
  --internal :32100
  --external :80
  --lifetime 120
  --server 127.0.0.1:5351
)

if [[ "$*" != "${EXPECTED_ARGV[*]}" ]]; then
  pt "TEST FAILED: unexpected argv"
  pt "Expected: ${EXPECTED_ARGV[*]}"
  pt "Actual  : $*"
  exit 111
fi

# Portable date invocation (GNU/BSD versions).
# In the format like `Sat Feb 13 22:23:42 2021`
# shellcheck disable=SC2251
! LEASE_DATE_END="$(LANG=C date -d 'in 120 seconds' +'%c' 2>/dev/null)"
BSD_DATE_EXITCODE="$?"
if [[ "$BSD_DATE_EXITCODE" -eq 0 ]]; then
  LEASE_DATE_END="$(LANG=C date -v+120S +'%c' 2>/dev/null)"
fi

pe ""
pe "  0s 000ms 000us INFO   : Found gateway ::ffff:192.168.0.1. Added as possible PCP server."
pe "  0s 000ms 018us INFO   : Found gateway fe80::abcd:abcd:abcd:abcd. Added as possible PCP server."
pe "  0s 000ms 024us INFO   : Added new flow(PCP server: ::ffff:192.168.0.1; Int. addr: [::ffff:192.168.0.2]:32100; Dest. addr: [::]:0; Key bucket: 27)"
pe "  0s 000ms 030us INFO   : Added new flow(PCP server: fe80::abcd:abcd:abcd:abcd; Int. addr: [fe80::a827:753b:6ee7:c79c]:32100; Dest. addr: [::]:0; Key bucket: 16)"
pe "  0s 000ms 033us INFO   : Initialized wait for result of flow: 27, wait timeout 1000 ms"
pe "  0s 000ms 038us INFO   : Pinging PCP server at address ::ffff:192.168.0.1"
pe "  0s 000ms 051us INFO   : Sent PCP MSG (flow bucket:27)"
pe "  0s 000ms 055us INFO   : Pinging PCP server at address fe80::abcd:abcd:abcd:abcd"
pe "  0s 000ms 065us INFO   : Sent PCP MSG (flow bucket:16)"
sleep 0.019
pe "  0s 019ms 564us INFO   : Received PCP packet from server at ::ffff:192.168.0.1, size 60, result_code 0, epoch 2228596"
pe "  0s 019ms 573us INFO   : Found matching flow 27 to received PCP message."
pe "  0s 019ms 760us INFO   : Received PCP packet from server at fe80::abcd:abcd:abcd:abcd, size 60, result_code 0, epoch 2228596"
pe "  0s 019ms 766us INFO   : Found matching flow 16 to received PCP message."
po ""
po "Flow signaling succeeded."
po "PCP Server IP        Prot Int. IP               port   Dst. IP               port   Ext. IP               port Res State Ends"
po "::ffff:192.168.0.1   TCP  ::ffff:192.168.0.2   32100   ::                       0   ::ffff:1.2.3.4        1024   0  succ $LEASE_DATE_END"
po "fe80::abcd:abcd:abcd:abcd TCP  fe80::abcd:abcd:abcd:ffff 32100   ::                       0   fe80::abcd:abcd:abcd:aaaa    80   0  succ $LEASE_DATE_END"
po ""
pe "  0s 019ms 860us INFO   : PCP server ::ffff:192.168.0.1 terminated."
pe "  0s 019ms 866us INFO   : PCP server fe80::abcd:abcd:abcd:abcd terminated."
pe ""

pt "OK"
