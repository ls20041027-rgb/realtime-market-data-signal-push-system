"""DEPRECATED: Filedata dispatch is no longer needed.

With filedata split into 4 separate Kafka topics at the data ingestion layer,
the stream engine reads each sub-type directly — no runtime dispatch required.

This file is kept as a stub to avoid import errors from any remaining references.
"""
