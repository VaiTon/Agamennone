package io.github.vaiton.agamennone

import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import io.github.vaiton.agamennone.model.Flags
import org.jetbrains.exposed.sql.*
import org.jetbrains.exposed.sql.transactions.TransactionManager
import org.jetbrains.exposed.sql.transactions.experimental.newSuspendedTransaction
import org.jetbrains.exposed.sql.transactions.transaction
import java.sql.Connection
import java.time.LocalDateTime

object FlagDatabase {

    fun init() {
        Database.connect("jdbc:sqlite:./data.db", "org.sqlite.JDBC")
        TransactionManager.manager.defaultIsolationLevel =
            Connection.TRANSACTION_SERIALIZABLE

        transaction {
            SchemaUtils.create(Flags)
        }
    }


    /**
     * @return the latest cycle, or null if the database is empty
     */
    suspend fun getMaxCycle(): Int? {
        return newSuspendedTransaction {
            val result = Flags.slice(Flags.sentCycle.max())
                .selectAll()
                .firstOrNull()

            result?.getOrNull<Int?>(Flags.sentCycle.max())
        }
    }

    /**
     * Update all the flags with a [Flag.receivedTime]
     * before [before] to [FlagStatus.SKIPPED].
     *
     * @return the number of flags that were skipped.
     */
    suspend fun skipOldFlags(before: LocalDateTime): Int {
        return newSuspendedTransaction {
            Flags.update({
                (Flags.receivedTime less before) and (Flags.status eq FlagStatus.QUEUED)
            }) {
                it[status] = FlagStatus.SKIPPED
            }
        }
    }

    /**
     * @return all the flags with [Flag.status] [FlagStatus.QUEUED]
     */
    suspend fun getQueuedFlags(): List<Flag> = newSuspendedTransaction {
        Flag.find { Flags.status eq FlagStatus.QUEUED }.toList()
    }


    suspend fun setFlagResponse(flag: Flag) {
        newSuspendedTransaction {
            Flag.find { Flags.flag eq flag.flag }.firstOrNull()?.apply {
                status = flag.status
                checkSystemResponse = flag.checkSystemResponse
            }
        }
    }

}