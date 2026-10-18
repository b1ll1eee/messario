#!/usr/bin/env bash
# Seed a running Nuntio with demo users, a bot, a group, a channel and a few
# messages. Good for screenshots and first-run demos. Idempotent enough to
# re-run (duplicate users are skipped).
#
#   BASE=http://localhost:8080 ADMIN=admin PASSWORD=admin12345 scripts/demo.sh
set -euo pipefail
BASE=${BASE:-http://localhost:8080}
ADMIN=${ADMIN:-admin}
PASSWORD=${PASSWORD:?set PASSWORD for the admin user}
J='Content-Type: application/json'
jar=$(mktemp)
trap 'rm -f "$jar"' EXIT

api() { curl -s -b "$jar" -c "$jar" -H "$J" "$@"; }
id_of() { python3 -c 'import sys,json;d=json.load(sys.stdin);print(d.get("id") or d.get("bot",{}).get("id") or 0)'; }

api -X POST "$BASE/api/auth/login" -d "{\"username\":\"$ADMIN\",\"password\":\"$PASSWORD\"}" >/dev/null

mk_user() { # username display department password
  api -X POST "$BASE/api/admin/users" -d "{\"username\":\"$1\",\"display_name\":\"$2\",\"department\":\"$3\",\"password\":\"$4\"}" | id_of
}
NAT=$(mk_user nat "ณัฐพล ศรีสุข" "พัฒนาระบบ" demo12345)
WAR=$(mk_user warisara "วริศรา ทองดี" "ฝ่ายขาย" demo12345)
SOM=$(mk_user somchai "สมชาย ใจดี" "ฝ่ายบุคคล" demo12345)
BOT=$(api -X POST "$BASE/api/admin/bots" -d '{"username":"deploy_bot","display_name":"Deploy Bot"}' | id_of)
BOT_TOKEN=$(api -X POST "$BASE/api/admin/bots/$BOT/token" | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')

GROUP=$(api -X POST "$BASE/api/chats" -d "{\"type\":\"group\",\"title\":\"ทีมพัฒนา\",\"member_ids\":[$NAT,$WAR,$BOT]}" | id_of)
CHANNEL=$(api -X POST "$BASE/api/chats" -d "{\"type\":\"channel\",\"title\":\"ประกาศบริษัท\",\"member_ids\":[$NAT,$WAR,$SOM,$BOT]}" | id_of)

say() { # cookie-jar-user chat body
  curl -s -b "$1" -H "$J" -X POST "$BASE/api/chats/$2/messages" -d "{\"body\":$(python3 -c 'import json,sys;print(json.dumps(sys.argv[1]))' "$3")}" >/dev/null
}
login_as() { local j; j=$(mktemp); curl -s -c "$j" -H "$J" -X POST "$BASE/api/auth/login" -d "{\"username\":\"$1\",\"password\":\"demo12345\"}" >/dev/null; echo "$j"; }
NJ=$(login_as nat); WJ=$(login_as warisara)

say "$NJ" "$GROUP" "เมื่อคืน merge PR ของระบบ invoice เข้า main แล้วนะครับ บ่ายนี้จะ deploy production"
say "$NJ" "$GROUP" "ใครมีอะไรค้างให้บอกก่อนเที่ยงครับ"
say "$WJ" "$GROUP" "ไม่มีค่ะ ของฝั่งหน้าบ้านเรียบร้อยหมดแล้ว"
curl -s -H "$J" -X POST "$BASE/bot/$BOT_TOKEN/sendMessage" -d "{\"chat_id\":$GROUP,\"text\":\"production v1.4.2 สำเร็จ · 1 นาที 48 วิ · a81f2c9 invoice: fix VAT rounding\"}" >/dev/null
say "$NJ" "$GROUP" "ขึ้นแล้วครับ ฝากทีมขายช่วยเช็กหน้า invoice ด้วย"
say "$WJ" "$GROUP" "เช็กแล้ว ผ่านค่ะ"

api -X POST "$BASE/api/chats/$CHANNEL/messages" -d '{"body":"วันหยุดชดเชย 23 ตุลาคม บริษัทหยุดชดเชยวันปิยมหาราชในวันพฤหัสบดีที่ 23 ตุลาคม พนักงานที่ต้องเข้าเวรให้ติดต่อหัวหน้าทีมล่วงหน้า"}' >/dev/null
curl -s -H "$J" -X POST "$BASE/bot/$BOT_TOKEN/sendMessage" -d "{\"chat_id\":$CHANNEL,\"text\":\"ปิดปรับปรุง VPN คืนวันศุกร์ 22:00 ถึง 23:30 งานที่ต้องใช้ VPN ให้ทำให้เสร็จก่อนเวลา\"}" >/dev/null

DIRECT=$(api -X POST "$BASE/api/chats/direct" -d "{\"user_id\":$NAT}" | id_of)
say "$NJ" "$DIRECT" "ขอไฟล์สัญญาฉบับล่าสุดหน่อยครับ"
rm -f "$NJ" "$WJ"
echo "demo ready: group=$GROUP channel=$CHANNEL direct=$DIRECT bot=$BOT"
echo "bot token saved to: ${DEMO_TOKEN_FILE:-/dev/null}"
[ -n "${DEMO_TOKEN_FILE:-}" ] && (umask 077; echo "$BOT_TOKEN" > "$DEMO_TOKEN_FILE")
exit 0
