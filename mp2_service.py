import asyncio
import sys
import random
import re
import time

if len(sys.argv) != 3:
    print("Usage: mp2_service.py <port_number> <tx_rate>")
    sys.exit(1)

port = int(sys.argv[1])
tx_rate = float(sys.argv[2])

connections = []
kill_connections = []
snap_event = asyncio.Event()


async def handle_connection(reader, writer):
    connect_line = await reader.readline()
    addr = writer.get_extra_info('peername')
    print(f"Received connection from {addr}")

    m = re.match(r'CONNECT\s+(.*)$', connect_line.decode(), re.I)
    if not m:
        print(f"Incorrect connect line format from {addr}: `{connect_line.strip()}'")
        writer.close()
    else:
        node_line = m.group(1)
        num_intros = min(len(connections), 3)
        intros = random.sample(connections, num_intros)
        for intro in intros:
            writer.write(f"INTRODUCE {intro}\n".encode())
        await writer.drain()

        try:
            connections.append(node_line)

            command_task = asyncio.ensure_future(reader.readline())
            event_task = asyncio.ensure_future(snap_event.wait())

            while True:

                timeout = random.expovariate(tx_rate / len(connections))
                ready, pending = await asyncio.wait([command_task, event_task], timeout=timeout,
                                                    return_when=asyncio.FIRST_COMPLETED)
                if not ready:  # timeout, send transaction
                    tx_time = time.time()
                    tx_id = format(random.randrange(2 ** 128), "032x")
                    tx_from = random.randrange(2 ** 20)
                    tx_to = random.randrange(2 ** 20)
                    tx_amount = random.randrange(2 ** 10)
                    tx = f"{tx_time:#.6f} {tx_id} {tx_from} {tx_to} {tx_amount}"
                    print(f"Sending transaction {tx} to {addr}")
                    writer.write(f"TRANSACTION {tx}\n".encode())
                    await writer.drain()
                else:
                    if command_task in ready:
                        command = command_task.result()
                        if True:
                            print(f"Unexpected command `{command.decode().strip()}' from {addr}, disconnecting")
                            break
                        # left here to remember when we start supporting commands
                        command_task = asyncio.ensure_future(reader.readline())
                    if event_task in ready:
                        if node_line in kill_connections:
                            kill_connections.remove(node_line)
                            print(f"Killing connection {node_line} @ {addr}")
                            writer.write(b"DIE\n")
                            await writer.drain()
                            break
                        event_task = asyncio.ensure_future(snap_event.wait())

        except ConnectionError:
            print(f"Error in connection with {addr}")
        finally:
            writer.close()
            connections.remove(node_line)


def handle_command():
    command = sys.stdin.readline()
    if command.lower().startswith("thanos"):
        # SNAP
        global kill_connections, snap_event
        kill_connections = set(random.sample(connections, len(connections) // 2))
        print(f"Killing {kill_connections}")
        snap_event.set()
        snap_event = asyncio.Event()
    else:
        print(f"Unknown command: `{command.strip()}''")


loop = asyncio.get_event_loop()
coro = asyncio.start_server(handle_connection, None, port, loop=loop)
server = loop.run_until_complete(coro)

# Serve requests until Ctrl+C is pressed
print('Serving on {}'.format(server.sockets[0].getsockname()))
loop.add_reader(sys.stdin, handle_command)
try:
    loop.run_forever()
except KeyboardInterrupt:
    pass

# Close the server
server.close()
loop.run_until_complete(server.wait_closed())
loop.close()
