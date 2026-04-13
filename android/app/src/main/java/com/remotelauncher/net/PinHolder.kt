package com.remotelauncher.net

import java.util.concurrent.atomic.AtomicReference

object PinHolder {
    private val current = AtomicReference<String?>(null)
    private val lastObserved = AtomicReference<String?>(null)

    fun setCurrent(pinHex: String?) {
        current.set(pinHex?.uppercase())
    }

    fun getCurrent(): String? = current.get()

    fun recordObserved(pinHex: String) {
        lastObserved.set(pinHex.uppercase())
    }

    fun consumeObserved(): String? = lastObserved.getAndSet(null)

    fun clear() {
        current.set(null)
        lastObserved.set(null)
    }
}
