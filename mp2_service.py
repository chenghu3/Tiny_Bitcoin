import asyncio
import sys
import random
import re
import time
from collections import defaultdict

if len(sys.argv) < 3 or len(sys.argv) > 4:
    print("Usage: mp2_service.py <port_number> <tx_rate> [block_rate]")
    sys.exit(1)

port = int(sys.argv[1])
tx_rate = float(sys.argv[2])
if len(sys.argv) > 3:
    block_rate = float(sys.argv[3])
else:
    block_rate = 1.0/60

connections = []
kill_connections = []
snap_event = asyncio.Event()
oracle = {}
balances = defaultdict(int)
balances[0] = 1000

async def handle_connection(reader, writer):
    global balances
    addr = writer.get_extra_info('peername')
    print(f"Received connection from {addr}")

    solving = None

    connect_line = await reader.readline()
    m = re.match(r'CONNECT\s+(.*)$', connect_line.decode(), re.I)
    if not m:
        print(f"Incorrect connect line format from {addr}: `{connect_line.strip()}'")
        writer.close()
    else:
        node_line = m.group(1)
        print(f"New node: {node_line}")
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
                tx_timeout = random.expovariate(tx_rate / len(connections))
                if solving:
                    block_timeout = random.expovariate(block_rate / len(connections))
                    timeout = min(tx_timeout, block_timeout)
                else:
                    timeout = tx_timeout
                ready, pending = await asyncio.wait([command_task, event_task], timeout=timeout,
                                                    return_when=asyncio.FIRST_COMPLETED)
                if not ready:  # timeout
                    if solving and block_timeout < tx_timeout:  # block solution
                        solution = format(random.randrange(2**256), "032x")
                        oracle[solving] = solution
                        print(f"Sending solution {solving} {solution} to {addr}")
                        writer.write(f"SOLVED {solving} {solution}\n".encode())
                        await writer.drain()
                        solving = None
                    else:
                        tx_time = time.time()
                        tx_id = format(random.randrange(2 ** 128), "032x")
                        tx_from = random.choice([ k for k,v in balances.items() if v != 0])
                        if len(balances) > 1:
                            tx_to = random.choice([ k for k in balances.keys() if k != tx_from])
                            if tx_to == 0:
                                tx_to = len(balances)
                        else:
                            tx_to = 1
                        tx_amount = random.randrange(1, balances[tx_from]+1)
                        if tx_from != 0:
                            balances[tx_from] -= tx_amount
                        balances[tx_to] += tx_amount
                        tx = f"{tx_time:#.6f} {tx_id} {tx_from} {tx_to} {tx_amount}"
                        print(f"Sending transaction {tx} to {addr}")
                        print(f"New balances: {tx_from}: {balances[tx_from]}, {tx_to}: {balances[tx_to]}")
                        writer.write(f"TRANSACTION {tx}\n".encode())
                        await writer.drain()
                else:
                    if command_task in ready:
                        command, *args = command_task.result().decode().strip().split()
                        if command.lower() == "solve" and len(args) == 1:
                            solving = args[0]
                            print(f"Solving {solving} for {addr}")
                        elif command.lower() == "verify" and len(args) == 2:
                            if oracle.get(args[0]) == args[1]:
                                print(f"Verification succeeded {args[0]} -> {args[1]} {addr}")
                                writer.write(f"VERIFY OK {args[0]} {args[1]}\n".encode())
                            else:
                                print(f"Verification failed {args[0]} -> {args[1]} {addr}")
                                writer.write(f"VERIFY FAIL {args[0]} {args[1]}\n".encode())
                            await writer.drain()
                        else:
                            print(f"Unexpected command `{command} {' '.join(args)}' from {addr}, disconnecting")
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
            event_task.cancel()
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