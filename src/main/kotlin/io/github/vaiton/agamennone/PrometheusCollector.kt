package io.github.vaiton.agamennone

import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.model.Flags
import io.prometheus.client.Collector
import io.prometheus.client.CounterMetricFamily
import io.prometheus.client.GaugeMetricFamily
import org.jetbrains.exposed.sql.SqlExpressionBuilder.eq
import org.jetbrains.exposed.sql.count
import org.jetbrains.exposed.sql.selectAll
import org.jetbrains.exposed.sql.transactions.transaction

