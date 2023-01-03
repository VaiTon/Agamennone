package io.github.vaiton.agamennone

import io.github.vaiton.agamennone.model.FlagStatus
import io.ktor.server.application.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import io.prometheus.client.CollectorRegistry
import io.prometheus.client.Counter
import io.prometheus.client.Gauge
import io.prometheus.client.Summary
import io.prometheus.client.exporter.common.TextFormat
import kotlin.time.DurationUnit

object Metrics {
    private val QUEUED_FLAGS: Gauge = Gauge.build(
        "flags_queued", "Number of queued flags"
    ).register()

    fun setQueuedFlags(value: Int) {
        QUEUED_FLAGS.set(value.toDouble())
    }

    private val TIMED_OUT_FLAGS: Counter = Counter.build(
        "flags_timed_out", "Number of skipped flags"
    ).register()

    fun incrementTimedOutFlags(inc: Long) {
        TIMED_OUT_FLAGS.inc(inc.toDouble())
    }

    private val FLAGS = mapOf<FlagStatus, Counter>(
        FlagStatus.ACCEPTED to Counter.build()
            .name("flags_accepted")
            .help("Number of sent flags")
            .labelNames("sploit", "team")
            .register(),

        FlagStatus.SKIPPED to Counter.build()
            .name("flags_skipped")
            .help("Number of skipped flags")
            .labelNames("sploit", "team")
            .register(),

        FlagStatus.REJECTED to Counter.build()
            .name("flags_rejected")
            .help("Numbers of rejected flags")
            .labelNames("sploit", "team")
            .register(),
    )

    fun incrementFlagStatus(status: FlagStatus, exploitName: String, team: String) {
        if (status == FlagStatus.QUEUED) return

        val counter = checkNotNull(FLAGS[status]) { "No counter for status $status" }
        counter.labels(exploitName, team).inc()

        // We set the queued flags at the start of the cycle. Here we decrease it every time a flag is submitted.
        QUEUED_FLAGS.dec()
    }

    private val SUBMITTER_LATENCY: Summary = Summary.Builder()
        .name("submitter_latency")
        .help("Submitter latency in milliseconds")
        .register()

    fun observeSubmitterLatency(latency: kotlin.time.Duration) {
        SUBMITTER_LATENCY.observe(latency.toDouble(DurationUnit.MILLISECONDS))
    }
}

fun Route.prometheus(
    url: String = "metrics",
    registry: CollectorRegistry = CollectorRegistry.defaultRegistry,
) {
    get(url) {
        val acceptHeader = call.request.header("Accept")
        val contentType = TextFormat.chooseContentType(acceptHeader)
        call.respondTextWriter {
            TextFormat.writeFormat(contentType, this, registry.metricFamilySamples())
        }
    }
}