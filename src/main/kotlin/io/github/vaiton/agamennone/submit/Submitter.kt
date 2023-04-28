package io.github.vaiton.agamennone.submit

import io.github.vaiton.agamennone.AgamennoneMetrics
import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.ConfigManager
import io.github.vaiton.agamennone.FlagDatabase
import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.model.Flags
import kotlinx.coroutines.delay
import mu.KotlinLogging
import org.jetbrains.exposed.sql.transactions.experimental.newSuspendedTransaction
import org.jetbrains.exposed.sql.update
import java.time.Duration
import java.time.LocalDateTime
import kotlin.time.Duration.Companion.seconds
import kotlin.time.toJavaDuration
import kotlin.time.toKotlinDuration

object Submitter {

    private val log = KotlinLogging.logger {}

    suspend fun loop(): Nothing {
        var cycle = FlagDatabase.getMaxCycle() ?: 0

        while (true) {
            cycle += 1
            val config = ConfigManager.config.value
            val submitStartTime = LocalDateTime.now()

            skipOldFlags(submitStartTime, config)

            val queuedFlags = FlagDatabase.getQueuedFlags()

            val toSubmit = queuedFlags.take(config.submissionFlagLimit)

            if (queuedFlags.isNotEmpty()) {
                log.info { "Submitting ${toSubmit.size} flags (out of ${queuedFlags.size} queued)." }
                submitFlags(toSubmit, config, cycle)
            }

            val submitEndTime = LocalDateTime.now()
            val loopTime = Duration.between(submitStartTime, submitEndTime).toKotlinDuration()

            // Update submitter latency
            AgamennoneMetrics.observeSubmitterLatency(loopTime)

            val submitPeriodInSeconds = config.submissionPeriod.seconds

            val sleepTime = (submitPeriodInSeconds - loopTime).coerceAtLeast(0.seconds)
            log.debug {
                "Submit loop finished in ${loopTime.inWholeMilliseconds}ms. " +
                        "Sleeping for ${sleepTime.inWholeSeconds}s."
            }
            if (sleepTime.isPositive()) {
                delay(sleepTime)
            }
        }
    }

    private suspend fun skipOldFlags(submitStartTime: LocalDateTime, config: Config) {
        log.debug { "Skipping old flags..." }
        // calculate the time before which flags should be skipped
        val skipTime = submitStartTime - config.flagLifetime.seconds.toJavaDuration()

        // skip flags that are older than the skip time
        val skipped = FlagDatabase.skipOldFlags(skipTime)

        // if any flags were skipped, log it and update the metrics
        log.info { "Skipped $skipped old flags." }
    }


    private suspend fun submitFlags(flags: List<Flag>, config: Config, cycle: Int) {
        val submitterProtocol = config.submissionProtocol
        val protocol = SubmissionProtocol.getProtocol(submitterProtocol)

        val submissions = flags.map { it.flag }

        val results = kotlin.runCatching {
            newSuspendedTransaction {
                protocol.submitFlags(submissions, config)
            }
        }.getOrElse {
            log.error(it) { "Error while submitting flags." }
            emptyList()
        }



        newSuspendedTransaction {
            results.forEach { flag ->
                Flags.update({ Flags.flag eq flag.flag }) {
                    it[status] = flag.status
                    it[sentCycle] = cycle
                }
                log.debug { "Submitted flag '${flag.flag}' with status ${flag.status}" }
            }
        }

        log.info {
            // We calculate the number in the log block so that we don't have to
            // do it if the log level is not info

            val acceptedCount = flags.count { it.status == FlagStatus.ACCEPTED }
            val rejectedCount = flags.count { it.status == FlagStatus.REJECTED }
            val skippedCount = flags.count { it.status == FlagStatus.SKIPPED }

            buildString {
                append("Submitted ")
                append(flags.size)
                append(" flags: ")
                if (acceptedCount > 0) {
                    append("accepted=$acceptedCount,")
                }
                if (rejectedCount > 0) {
                    append("|rejected=$rejectedCount,")
                }
                if (skippedCount > 0) {
                    append("|skipped=$skippedCount")
                }
            }
        }
    }

}