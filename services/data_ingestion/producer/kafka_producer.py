"""
Kafka 生产者线程：从内部队列批量 drain 消息并发送到 Kafka。

拆分动机：把队列与生产者线程独立成一个子包（对齐 stream_engine/producer/），
让 DLL 回调 / 解析器 / 主链路都只依赖一个稳定的入队 API，而不必与 Producer
实现耦合。队列与停机信号（data_queue / stop_event）也放在这里，保证"谁启动
线程、谁拥有状态"原则。
"""

import queue
import threading

import orjson
from confluent_kafka import Producer

from config import settings

data_queue = queue.Queue(maxsize=settings.service.queue_maxsize)

stop_event = threading.Event()


def _kafka_delivery_report(err, msg):
    if err is not None:
        print(f"[ERROR] kafka delivery failed: {err}")


def kafka_producer_worker():
    producer = Producer({
        "bootstrap.servers": settings.kafka.bootstrap_servers,
        "client.id": settings.kafka.client_id,
        "batch.size": settings.kafka.batch_size,
        "linger.ms": settings.kafka.linger_ms,
        "compression.type": settings.kafka.compression_type,
        "queue.buffering.max.messages": settings.kafka.queue_buffering_max_messages,
        "queue.buffering.max.kbytes": settings.kafka.buffer_memory // 1024,
    })
    print(
        f"[INFO] kafka producer started, "
        f"broker={settings.kafka.bootstrap_servers}"
    )

    batch_size = settings.service.batch_drain_size

    try:
        while not stop_event.is_set():
            batch = []
            try:
                batch.append(data_queue.get(timeout=1))
            except queue.Empty:
                producer.poll(0)
                continue

            for _ in range(batch_size - 1):
                try:
                    batch.append(data_queue.get_nowait())
                except queue.Empty:
                    break

            for queue_item in batch:
                topic = queue_item.get("topic", settings.kafka.topic_report)
                message = queue_item.get("message", {})
                key_str = message.get("symbol") or ""
                key_bytes = key_str.encode("utf-8") if key_str else None
                try:
                    encoded = orjson.dumps(message, default=str)
                    producer.produce(topic, key=key_bytes, value=encoded, callback=_kafka_delivery_report)
                except BufferError:
                    producer.poll(1.0)
                    try:
                        producer.produce(topic, key=key_bytes, value=encoded, callback=_kafka_delivery_report)
                    except Exception as exc:
                        print(f"[ERROR] kafka send failed after flush: {exc}")
                except Exception as exc:
                    print(f"[ERROR] kafka send failed: {exc}")

            producer.poll(0)

            for _ in batch:
                data_queue.task_done()
    finally:
        producer.flush(10)
