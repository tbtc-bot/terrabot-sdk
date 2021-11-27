package config

import "github.com/spf13/viper"

func SetDefaultValues() error {
	// Set default values
	viper.SetDefault("applicationMode", "prod")
	viper.SetDefault("gcpFirebaseAdminServiceAccount", "firebase-admin.json")
	viper.SetDefault("rabbitMqHostnames", "localhost")
	viper.SetDefault("rabbitMqPort", "5672")
	viper.SetDefault("RABBITMQ_USER", "lab")
	viper.SetDefault("RABBITMQ_PASSWORD", "lab")

	// tracing
	viper.SetDefault("tracing", false)
	viper.SetDefault("GOOGLE_CLOUD_PROJECT", "")
	viper.SetDefault("zipkinUrl", "http://localhost:9411/api/v2/spans")
	viper.SetDefault("tracingSampleRatio", "1") // between 0.0 and 1.0

	// exchange
	viper.SetDefault("rabbitMqBinanceFuturesUserDataExchange", "binance.futures.userData.mdx") // mdx =  message duplication exchange

	// routing keys
	viper.SetDefault("rabbitMqCommandsRoutingKey", "binance.futures.command.worker")
	viper.SetDefault("rabbitMqEventsRoutingKey", "binance.futures.event.userData")

	// queues
	viper.SetDefault("rabbitMqCommandsQueueName", "binance_futures_command_worker_execute_qq")
	viper.SetDefault("rabbitMqEventsQueueName", "binance_futures_event_userData_trade_qq")

	viper.SetDefault("rabbitMqVhost", "lab")

	viper.SetDefault("redisHostnames", "localhost:6379")
	viper.SetDefault("redisPassword", "")
	viper.SetDefault("POD_NAME", "local")

	// load configuration firebase, rabbitmq
	viper.SetConfigName("config")       // name of config file (without extension)
	viper.SetConfigType("yaml")         // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")            // optionally look for config in the working directory
	viper.AddConfigPath("/configmaps/") // k8s config map as file
	viper.AutomaticEnv()

	return viper.ReadInConfig() // Find and read the config file
}
