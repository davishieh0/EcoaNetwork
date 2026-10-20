"""
Generated stubs for the node-facing gRPC contract.
"""

# --- IMPORTS ---
from src.grpc.proto import node_api_pb2

import sys


# --- CODE ---
# protoc emits a flat `import node_api_pb2`; this makes it resolve in-package
sys.modules.setdefault('node_api_pb2', node_api_pb2)
