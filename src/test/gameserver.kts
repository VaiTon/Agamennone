import java.net.ServerSocket
import java.time.LocalTime
import kotlin.random.Random

val server = ServerSocket(663)

println("Starting mock gameserver...")
val seed = System.currentTimeMillis()
val random = Random(seed)
while (true) {
    kotlin.runCatching {
        val socket = server.accept()
        val input = socket.getInputStream().bufferedReader()
        val output = socket.getOutputStream().writer()

        output.write("Hello, world! THis is the banner!!!!\n\n")
        output.flush()

        while (true) {
            val line = input.readLine()
            println("Received: $line")

            val randomInt = random.nextInt(3)
            val status = when (randomInt) {
                0 -> "OK sei bello"
                1 -> "DUP noooo duplicato"
                2 -> "INV invalido proprio"
                else -> error("This should not happen")
            }
            output.write("$line $status\n")
            output.flush()
        }
    }
}