package io.github.vaiton.agamennone

import io.github.vaiton.agamennone.model.Flag
import io.github.vaiton.agamennone.model.FlagStatus
import org.bson.conversions.Bson
import org.litote.kmongo.*
import org.litote.kmongo.coroutine.CoroutineCollection
import org.litote.kmongo.coroutine.coroutine
import org.litote.kmongo.reactivestreams.KMongo
import java.time.LocalDateTime

object FlagDatabase {

    private lateinit var collection: CoroutineCollection<Flag>

    fun init() {
        val client = KMongo.createClient().coroutine
        val database = client.getDatabase("agamennone")
        collection = database.getCollection()
    }

    /**
     * @return the number of flags inserted
     */
    suspend fun addFlags(flags: List<Flag>): Int {
        val result = collection.insertMany(flags)
        return result.insertedIds.count()
    }

    /**
     * @return the latest cycle, or null if the database is empty
     */
    suspend fun getMaxCycle(): Int? {
        return collection.find()
            .descendingSort(Flag::sentCycle)
            .first()
            ?.sentCycle
    }

    /**
     * Update all the flags with a [Flag.receivedTime]
     * before [before] to [FlagStatus.SKIPPED].
     *
     * @return the number of flags that were skipped.
     */
    suspend fun skipOldFlags(before: LocalDateTime): Long {
        val update = collection.updateMany(
            and(
                Flag::receivedTime lt before,
                Flag::status eq FlagStatus.QUEUED
            ),
            Flag::status setTo FlagStatus.SKIPPED
        )
        return update.modifiedCount
    }

    /**
     * @return all the flags with [Flag.status] [FlagStatus.QUEUED]
     */
    suspend fun getQueuedFlags(): List<Flag> = collection
        .find(Flag::status eq FlagStatus.QUEUED)
        .toList()


    suspend fun setFlagResponse(flag: Flag) {
        collection.updateOne(
            Flag::flag eq flag.flag,
            set(
                Flag::status setTo flag.status,
                Flag::checkSystemResponse setTo flag.checkSystemResponse
            )
        )
    }

    suspend fun getFlags(filter: Bson, limit: Int? = null): List<Flag> {
        return collection.find(filter)
            .apply { if (limit != null) limit(limit) }
            .toList()
    }

}