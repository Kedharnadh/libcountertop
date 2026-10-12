import struct
import sys

import numpy as np
import onnx
from onnx import numpy_helper

LAYERS = [
    "encoder.0.reparam_conv.weight",
    "encoder.0.reparam_conv.bias",
    "encoder.1.reparam_conv.weight",
    "encoder.1.reparam_conv.bias",
    "encoder.2.reparam_conv.weight",
    "encoder.2.reparam_conv.bias",
    "encoder.3.reparam_conv.weight",
    "encoder.3.reparam_conv.bias",
    "decoder.rnn.weight_ih",
    "decoder.rnn.weight_hh",
    "decoder.rnn.bias_ih",
    "decoder.rnn.bias_hh",
    "decoder.decoder.2.weight",
    "decoder.decoder.2.bias",
]

model = onnx.load(sys.argv[1])
branch = [a.g for a in [n for n in model.graph.node if n.op_type == "If"][0].attribute if a.name == "then_branch"][0]
found = {}
for node in branch.node:
    if node.op_type == "Constant":
        found[node.output[0].split("__")[-1]] = numpy_helper.to_array(node.attribute[0].t)
with open(sys.argv[2], "wb") as out:
    for name in LAYERS:
        values = np.ascontiguousarray(found[name], dtype="<f4").ravel()
        out.write(struct.pack(f"<{values.size}f", *values))
