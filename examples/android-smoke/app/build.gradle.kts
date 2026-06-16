plugins {
    id("com.android.application")
}

android {
    namespace = "com.thecitadelx.slipstream.smoke"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.thecitadelx.slipstream.smoke"
        minSdk = 23
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"
    }
}

dependencies {
    implementation(files("libs/libslipstream.aar"))
}
