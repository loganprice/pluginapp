#!/usr/bin/env python3

import argparse
import grpc
import time
from concurrent import futures

import plugin_pb2
import plugin_pb2_grpc
from grpc_health.v1 import health
from grpc_health.v1 import health_pb2
from grpc_health.v1 import health_pb2_grpc

class MultiplyPlugin(plugin_pb2_grpc.PluginServicer):
    def GetInfo(self, request, context):
        return plugin_pb2.PluginInfo(
            name="python-multiply",
            version="1.0.0",
            description="A Python multiplication plugin",
            parameter_specs={
                "num1": plugin_pb2.ParamSpec(
                    name="num1",
                    description="First number to multiply",
                    required=True,
                    default_value="2",
                    type="float"
                ),
                "num2": plugin_pb2.ParamSpec(
                    name="num2",
                    description="Second number to multiply",
                    required=True,
                    default_value="3",
                    type="float"
                )
            }
        )

    def Execute(self, request, context):
        try:
            # Parse parameters
            num1 = float(request.params.get("num1", "2"))
            num2 = float(request.params.get("num2", "3"))

            # Initial progress
            yield plugin_pb2.ExecuteOutput(
                progress=plugin_pb2.Progress(
                    stage="Starting",
                    percent_complete=0,
                    current_step=1,
                    total_steps=4
                )
            )

            # Initial message
            yield plugin_pb2.ExecuteOutput(
                output=f"Starting multiplication of {num1} and {num2}..."
            )
            time.sleep(1)

            # Processing progress
            yield plugin_pb2.ExecuteOutput(
                progress=plugin_pb2.Progress(
                    stage="Processing",
                    percent_complete=25,
                    current_step=2,
                    total_steps=4
                )
            )

            # Calculate result
            result = num1 * num2

            # Calculation progress
            yield plugin_pb2.ExecuteOutput(
                progress=plugin_pb2.Progress(
                    stage="Calculating",
                    percent_complete=75,
                    current_step=3,
                    total_steps=4
                )
            )
            time.sleep(1)

            # Final progress
            yield plugin_pb2.ExecuteOutput(
                progress=plugin_pb2.Progress(
                    stage="Finalizing",
                    percent_complete=100,
                    current_step=4,
                    total_steps=4
                )
            )

            # Final result
            yield plugin_pb2.ExecuteOutput(
                output=f"\nResult: {num1} × {num2} = {result}"
            )

        except ValueError as e:
            yield plugin_pb2.ExecuteOutput(
                error=plugin_pb2.Error(
                    code="INVALID_PARAMETERS",
                    message=str(e)
                )
            )
        except Exception as e:
            yield plugin_pb2.ExecuteOutput(
                error=plugin_pb2.Error(
                    code="EXECUTION_ERROR",
                    message=str(e)
                )
            )

    def ReportExecutionSummary(self, request, context):
        return plugin_pb2.SummaryResponse(
            plugin_name="python-multiply",
            start_time=request.start_time,
            end_time=request.end_time,
            duration=(request.end_time - request.start_time) / 1e6,  # Convert to milliseconds
            success=request.success,
            error=request.error,
            metadata=request.metadata,
            metrics=request.metrics
        )

def serve(port):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    plugin_pb2_grpc.add_PluginServicer_to_server(MultiplyPlugin(), server)

    # Add health service
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)

    # Try to bind to the port
    try:
        server.add_insecure_port(f'localhost:{port}')
    except RuntimeError:
        print(f"Failed to bind to port {port}, trying 127.0.0.1")
        server.add_insecure_port(f'127.0.0.1:{port}')

    server.start()
    print(f"Plugin server running on port {port}")
    server.wait_for_termination()

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, required=True, help='Port to listen on')
    args = parser.parse_args()
    serve(args.port) 