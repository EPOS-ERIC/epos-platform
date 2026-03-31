import com.github.jengelman.gradle.plugins.shadow.tasks.ShadowJar
import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    kotlin("jvm") version "2.3.0"
    application
    id("com.gradleup.shadow") version "9.0.0"
}

group = "org.epos"
version = "1.0-SNAPSHOT"
description = "Kotlin application for managing RDF lifecycle in GraphDB with versioning and PROV-O"

val rdf4jVersion            = "5.1.6"
val ktorVersion             = "3.0.3"
val coroutinesVersion       = "1.9.0"
val jacksonVersion          = "2.15.3"
val slf4jVersion            = "2.0.9"
val ktorOpenApiToolsVersion = "5.4.0"
val serializationVersion    = "1.6.2"
val koinVersion             = "3.5.6"
val junitVersion            = "5.10.0"
val testcontainersVersion   = "1.20.4"
val kotestVersion           = "5.8.0"

java {
    toolchain {
        languageVersion.set(JavaLanguageVersion.of(17))
    }
}

kotlin {
    jvmToolchain(17)
    compilerOptions {
        jvmTarget.set(JvmTarget.JVM_17)
    }
}

application {
    mainClass.set("org.epos.ApplicationKt")
}

dependencies {
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core:$coroutinesVersion")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-jdk8:$coroutinesVersion")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:$serializationVersion")

    implementation("org.eclipse.rdf4j:rdf4j-repository-api:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-repository-sparql:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-model:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-model-api:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-model-vocabulary:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-rio-api:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-rio-turtle:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-rio-jsonld:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-queryresultio-api:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-queryresultio-sparqljson:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-queryresultio-sparqlxml:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-queryresultio-text:$rdf4jVersion")
    implementation("org.eclipse.rdf4j:rdf4j-sparqlbuilder:$rdf4jVersion")

    implementation("io.ktor:ktor-server-core-jvm:$ktorVersion")
    implementation("io.ktor:ktor-server-netty-jvm:$ktorVersion")
    implementation("io.ktor:ktor-server-content-negotiation-jvm:$ktorVersion")
    implementation("io.ktor:ktor-serialization-jackson-jvm:$ktorVersion")
    implementation("io.ktor:ktor-server-cors-jvm:$ktorVersion")
    implementation("io.ktor:ktor-server-status-pages-jvm:$ktorVersion")
    implementation("io.ktor:ktor-server-call-logging-jvm:$ktorVersion")
    implementation("io.ktor:ktor-server-html-builder-jvm:$ktorVersion")
    implementation("io.ktor:ktor-client-core-jvm:$ktorVersion")
    implementation("io.ktor:ktor-client-cio-jvm:$ktorVersion")
    implementation("io.ktor:ktor-client-content-negotiation-jvm:$ktorVersion")

    implementation("io.github.smiley4:ktor-openapi:$ktorOpenApiToolsVersion")
    implementation("io.github.smiley4:ktor-swagger-ui:$ktorOpenApiToolsVersion")
    implementation("io.github.smiley4:ktor-redoc:$ktorOpenApiToolsVersion")

    implementation("com.fasterxml.jackson.module:jackson-module-kotlin:$jacksonVersion")
    implementation("com.fasterxml.jackson.datatype:jackson-datatype-jsr310:$jacksonVersion")
    implementation("com.fasterxml.jackson.core:jackson-databind:$jacksonVersion")
    implementation("com.fasterxml.jackson.dataformat:jackson-dataformat-yaml:$jacksonVersion")

    implementation("org.slf4j:slf4j-simple:$slf4jVersion")

    implementation("io.insert-koin:koin-core:$koinVersion")
    implementation("io.insert-koin:koin-ktor:$koinVersion")
    implementation("io.insert-koin:koin-logger-slf4j:$koinVersion")

    testImplementation(kotlin("test"))
    testImplementation("org.junit.jupiter:junit-jupiter-api:$junitVersion")
    testRuntimeOnly("org.junit.jupiter:junit-jupiter-engine:$junitVersion")
    testImplementation("org.testcontainers:testcontainers:$testcontainersVersion")
    testImplementation("io.kotest:kotest-assertions-core-jvm:$kotestVersion")
    testImplementation("io.mockk:mockk:1.13.8")
    testImplementation("io.ktor:ktor-server-test-host-jvm:$ktorVersion")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:$coroutinesVersion")
    testRuntimeOnly("ch.qos.logback:logback-classic:1.4.14")
}

tasks.test {
    useJUnitPlatform()
}

tasks.named<Jar>("jar") {
    archiveClassifier.set("plain")
}

tasks.named<ShadowJar>("shadowJar") {
    archiveClassifier.set("")
    mergeServiceFiles()
    exclude("META-INF/*.SF", "META-INF/*.DSA", "META-INF/*.RSA")
    manifest {
        attributes["Main-Class"] = application.mainClass.get()
    }
}

tasks.assemble {
    dependsOn(tasks.named("shadowJar"))
}
