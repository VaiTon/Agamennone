package io.github.vaiton.agamennone

import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.model.Flags
import io.ktor.server.application.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import io.prometheus.client.Collector
import io.prometheus.client.CollectorRegistry
import io.prometheus.client.CounterMetricFamily
import io.prometheus.client.Summary
import io.prometheus.client.exporter.common.TextFormat
import org.jetbrains.exposed.sql.alias
import org.jetbrains.exposed.sql.count
import org.jetbrains.exposed.sql.selectAll
import org.jetbrains.exposed.sql.transactions.transaction
import kotlin.time.DurationUnit

object AgamennoneMetrics {
    private val SUBMITTER_LATENCY: Summary = Summary.Builder()
        .name("submitter_latency")
        .help("Submitter latency in milliseconds")
        .create()

    fun observeSubmitterLatency(latency: kotlin.time.Duration) {
        SUBMITTER_LATENCY.observe(latency.toDouble(DurationUnit.MILLISECONDS))
    }
}

private object AgamennoneCollector : Collector() {

    override fun collect(): List<MetricFamilySamples> {
        val metrics = mutableListOf<MetricFamilySamples>()
        metrics += flags()
        return metrics
    }


    private fun flags(): List<MetricFamilySamples> {
        val labelNames = listOf("sploit", "team")
        val acceptedFlags = CounterMetricFamily(
            "accepted_flags",
            "Number of flags that were accepted",
            labelNames
        )
        val rejectedFlags = CounterMetricFamily(
            "rejected_flags",
            "Number of flags that were rejected",
            labelNames
        )
        val skippedFlags = CounterMetricFamily(
            "skipped_flags",
            "Number of flags that were skipped",
            labelNames
        )
        val queuedFlags = CounterMetricFamily(
            "queued_flags",
            "Number of flags that are queued",
            labelNames
        )

        transaction {
            val countCol = Flags.flag.count().alias("count")
            val teamCol = Flags.team
            val sploitCol = Flags.sploit
            val statusCol = Flags.status

            val flags = Flags.slice(countCol, teamCol, statusCol, sploitCol)
                .selectAll()
                .groupBy(teamCol, statusCol, sploitCol)
                .toList()

            flags.forEach {
                val labels = listOf(it[sploitCol], it[teamCol])
                val count = it[countCol].toDouble()

                when (it[statusCol]) {
                    FlagStatus.ACCEPTED -> acceptedFlags.addMetric(labels, count)
                    FlagStatus.REJECTED -> rejectedFlags.addMetric(labels, count)
                    FlagStatus.SKIPPED -> skippedFlags.addMetric(labels, count)
                    FlagStatus.QUEUED -> queuedFlags.addMetric(labels, count)
                }
            }
        }

        return listOf(acceptedFlags, rejectedFlags, skippedFlags)
    }
}

fun Route.prometheus() {
    val registry = CollectorRegistry.defaultRegistry
    registry.register(AgamennoneCollector)

    get("/metrics") {
        val acceptHeader = call.request.header("Accept")
        val contentType = TextFormat.chooseContentType(acceptHeader)
        call.respondTextWriter {
            TextFormat.writeFormat(contentType, this, registry.metricFamilySamples())
        }
    }
}
