#!/bin/sh
# Deklarasi topologi bersama (DLX + DLQ). Idempoten: aman dijalankan berulang.
# Membaca kredensial dari .env di root workspace.
set -e
cd "$(dirname "$0")/.."
set -a; . ./.env; set +a

RMQ="docker exec medcore-message-broker rabbitmqadmin -u $RABBITMQ_DEFAULT_USER -p $RABBITMQ_DEFAULT_PASS -V $RABBITMQ_DEFAULT_VHOST"

# Exchange utama (sama persis dengan yang dideklarasikan publisher)
$RMQ declare exchange name=medcore.clinical.events type=topic durable=true

# DLX + DLQ Pharmacy (sesuai modul)
$RMQ declare exchange name=medcore.deadletter.exchange type=fanout durable=true
$RMQ declare queue name=pharmacy.dlq durable=true
$RMQ declare binding source=medcore.deadletter.exchange destination=pharmacy.dlq

# DLX + DLQ Billing (terpisah agar pesan Pharmacy tidak ikut masuk ke billing.dlq)
$RMQ declare exchange name=medcore.deadletter.billing.exchange type=fanout durable=true
$RMQ declare queue name=billing.dlq durable=true
$RMQ declare binding source=medcore.deadletter.billing.exchange destination=billing.dlq

echo "Topologi RabbitMQ siap."
