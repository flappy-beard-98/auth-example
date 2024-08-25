package com.example.game_client

import android.annotation.SuppressLint
import android.content.Context
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.util.Log
import android.webkit.CookieManager
import android.webkit.JavascriptInterface
import android.webkit.ValueCallback
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import okhttp3.*
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.io.IOException

class MainActivity : ComponentActivity() {

    private var authSessionId = ""
    private val client = OkHttpClient.Builder()
        .build()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            AuthenticationScreen()
        }
    }

    @Composable
    fun AuthenticationScreen() {
        var sessionId by remember { mutableStateOf<String?>(null) }
        var authResult by remember { mutableStateOf("") }
        var authFrontendUrl by remember { mutableStateOf<String?>(null) }
        var showWebView by remember { mutableStateOf(false) }
        var url by remember { mutableStateOf("http://localhost:35183/auth") }

        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(16.dp),
            verticalArrangement = Arrangement.Center
        ) {
            UrlInputField(url) { url = it }

            Spacer(modifier = Modifier.height(16.dp))

            LoginButton {
                initiateAuthentication(url) { session, authUrl ->
                    sessionId = session
                    authFrontendUrl = authUrl
                    showWebView = true
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            Text(text = "Session ID: ${sessionId.orEmpty()}")

            Spacer(modifier = Modifier.height(16.dp))

            Text(text = "Auth Result: $authResult")

            Spacer(modifier = Modifier.height(16.dp))

            if (showWebView && authFrontendUrl != null) {
                WebViewContainer(authFrontendUrl!!) { login, otp ->
                    authResult = "User authenticated: Login=$login, OTP=$otp"
                    showWebView = false
                }
            }
        }
    }

    @Composable
    fun UrlInputField(url: String, onValueChange: (String) -> Unit) {
        BasicTextField(
            value = url,
            onValueChange = onValueChange,
            modifier = Modifier
                .border(2.dp, Color.Blue)
                .padding(8.dp),
            singleLine = true
        )
    }

    @Composable
    fun LoginButton(onClick: () -> Unit) {
        Button(onClick = onClick) {
            Text("Login")
        }
    }

    private fun initiateAuthentication(url: String, onSuccess: (String?, String?) -> Unit) {
        val request = Request.Builder()
            .url(Utils.fixUrlHost(url))
            .post(ByteArray(0).toRequestBody())
            .build()

        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                Log.e("MainActivity", "Failed to initiate authentication", e)
            }

            override fun onResponse(call: Call, response: Response) {
                if (response.isSuccessful) {
                    response.body?.string()?.let { responseBody ->
                        val json = JSONObject(responseBody)
                        val sessionId = json.getString("sessionId")
                        val authFrontendUrl = json.getString("authFrontendUrl")

                        onSuccess(sessionId, Utils.fixUrlHost(authFrontendUrl))
                    }
                } else {
                    Log.e(
                        "MainActivity",
                        "Failed to initiate authentication, response code: ${response.code}"
                    )
                }
            }
        })
    }

    @SuppressLint("SetJavaScriptEnabled")
    @Composable
    fun WebViewContainer(url: String, onResult: (String, String) -> Unit) {
        val context = LocalContext.current
        AndroidView(factory = {
            WebView(context).apply {
                settings.javaScriptEnabled = true
                webViewClient = CustomWebViewClient(onResult)
                loadUrl(Utils.fixUrlHost(url))
            }
        }, modifier = Modifier.fillMaxSize())
    }

    private inner class CustomWebViewClient(
        private val onResult: (String, String) -> Unit
    ) : WebViewClient() {

        override fun onPageFinished(view: WebView?, url: String?) {
            super.onPageFinished(view, url)
            Utils.fixHtmlHost(view)
            Utils.cookieNotification(view) { r ->
                authSessionId = r
            }

            url?.let {
                if (it.contains("login") && it.contains("otp")) {
                    val uri = Uri.parse(Utils.fixUrlHost(it))
                    val login = uri.getQueryParameter("login")
                    val otp = uri.getQueryParameter("otp")
                    if (login != null && otp != null) {
                        onResult(login, otp)
                    }
                }
            }
        }
    }

    companion object Utils {
        private val isEmulator: Boolean
            get() = listOf(
                Build.BRAND.startsWith("generic"),
                Build.DEVICE.startsWith("generic"),
                Build.FINGERPRINT.startsWith("generic"),
                Build.FINGERPRINT.startsWith("unknown"),
                Build.HARDWARE.contains("goldfish"),
                Build.HARDWARE.contains("ranchu"),
                Build.MODEL.contains("google_sdk"),
                Build.MODEL.contains("Emulator"),
                Build.MODEL.contains("Android SDK built for x86"),
                Build.MANUFACTURER.contains("Genymotion"),
                Build.PRODUCT.contains("sdk_google"),
                Build.PRODUCT.contains("google_sdk"),
                Build.PRODUCT.contains("sdk"),
                Build.PRODUCT.contains("sdk_x86"),
                Build.PRODUCT.contains("sdk_gphone64_arm64"),
                Build.PRODUCT.contains("vbox86p"),
                Build.PRODUCT.contains("emulator"),
                Build.PRODUCT.contains("simulator")
            ).any { it }

        fun fixUrlHost(url: String): String {
            return if (isEmulator) {
                val uri = Uri.parse(url)
                uri
                    .buildUpon()
                    .encodedAuthority("10.0.2.2:${uri.port}")
                    .build()
                    .toString()
            } else {
                url
            }
        }

        fun fixHtmlHost(view: WebView?) {
            if (!isEmulator) return
            view?.evaluateJavascript(
                """
                (function() { 
                        var html = document.documentElement.outerHTML;
                        html = html.replace(/localhost/g, '10.0.2.2');
                        document.open();
                        document.write(html);
                        document.close();
                        })();
                        """,
                null
            )
        }

        fun cookieNotification(view: WebView?, resultCallback: ValueCallback<String>) {
            view?.evaluateJavascript("""
                (function() { 
                    function getCookie(name) {
                        var value = "; " + document.cookie;
                        var parts = value.split("; " + name + "=");
                        if (parts.length == 2) return parts.pop().split(";").shift();
                    }
                    return getCookie('auth_session_id'); 
                })()
                    """,
                resultCallback
            )
        }
    }
}
