"""DEPRECATED: Normalize operators are no longer needed.

With fully normalized messages from the data ingestion layer, the stream engine
reads typed JSON directly via Pathway's native Kafka connector. All normalization
(symbol, precision, time, array expansion) is done at ingestion time.

This file is kept as a stub to avoid import errors from any remaining references.
"""
