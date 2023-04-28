package io.github.vaiton.agamennone.submit

import io.github.vaiton.agamennone.AgamennoneMetrics
import io.github.vaiton.agamennone.Config
import io.github.vaiton.agamennone.ConfigManager
import io.github.vaiton.agamennone.storage.FlagDatabase
import io.github.vaiton.agamennone.storage.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.storage.Flags
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
        val protocol = SubmissionProtocol.createProtocol(submitterProtocol)

        val submissions = flags.map { it.flag }

        kotlin.runCatching {
            protocol.submitFlags(submissions, config)
                .collect { (flag, status) -> setFlagStatus(flag, status, cycle) }
        }.getOrElse {
            log.error(it) { "Error while submitting flags." }
        }
    }

    private suspend fun setFlagStatus(
        flag: String,
        status: FlagStatus,
        cycle: Int,
    ) = newSuspendedTransaction {
        Flags.update({ Flags.flag eq flag }) { it ->
            it[Flags.status] = status
            it[sentCycle] = cycle
        }
        log.debug { "Submitted flag '${flag}' with status $status" }
    }

}