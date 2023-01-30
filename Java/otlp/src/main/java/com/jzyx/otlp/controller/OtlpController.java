package com.jzyx.otlp.controller;

import jakarta.servlet.http.HttpServletResponse;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.io.IOException;

@RestController
@RequestMapping("/")
public class OtlpController {
    @RequestMapping("/")
    public void index(HttpServletResponse response) throws IOException {
        response.sendRedirect("/test");
    }

    @RequestMapping("/test")
    public String test(){
        return "Java Spring Return OK!";
    }
}
