const PasswordHasher = (function () {
    async function sha256(message) {
        const encoder = new TextEncoder()
        const data = encoder.encode(message)
        const hashBuffer = await crypto.subtle.digest('SHA-256', data)
        const hashArray = Array.from(new Uint8Array(hashBuffer))
        return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
    }

    async function registerFormWithPassword(formId, passwordId, hashedPasswordId) {
        document.getElementById(formId).addEventListener("submit", async function (event) {
            event.preventDefault()
            const passwordInput = document.getElementById(passwordId)
            const hashedPasswordInput = document.getElementById(hashedPasswordId)
            const hashedPassword = await sha256(passwordInput.value)
            hashedPasswordInput.value = hashedPassword
            event.target.submit()
        });
    }

    return {
        registerFormWithPassword: registerFormWithPassword,
    }
})()