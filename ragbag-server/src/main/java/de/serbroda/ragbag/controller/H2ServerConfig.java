package de.serbroda.ragbag.controller;

import java.sql.SQLException;
import javax.sql.DataSource;
import liquibase.integration.spring.SpringLiquibase;
import org.h2.tools.Server;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.DependsOn;
import org.springframework.context.annotation.Profile;

@Configuration
@Profile("h2")
public class H2ServerConfig {

    @Value("${spring.liquibase.change-log}")
    private String changeLogPath;

    @Bean(name = "h2TcpServer", initMethod = "start", destroyMethod = "stop")
    public Server h2TcpServer() throws SQLException {
        return Server.createTcpServer("-tcp", "-tcpPort", "9092", "-ifNotExists", "-tcpAllowOthers");
    }

    @Bean
    @DependsOn("h2TcpServer")
    public SpringLiquibase liquibase(DataSource dataSource) {
        SpringLiquibase liquibase = new SpringLiquibase();
        liquibase.setDataSource(dataSource);
        liquibase.setChangeLog(changeLogPath);
        return liquibase;
    }
}
