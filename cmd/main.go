package main

import (
    "fmt"
    "log"
    "queue_facts/internal/factbuffer"
)

func main() {
    log.Println("Запуск буфера фактов...")

    buffer := factbuffer.NewFactBuffer(5000)

    // Отправляем 10 тестовых фактов через буфер
    for i := 1; i <= 10; i++ {
        fact := factbuffer.Fact{
            PeriodStart:         "2024-12-01",
            PeriodEnd:           "2024-12-31",
            PeriodKey:           "month",
            IndicatorToMoID:     "227373",
            IndicatorToMoFactID: "0",
            Value:               fmt.Sprintf("%d", i),
            FactTime:            "2024-12-31",
            IsPlan:              "0",
            AuthUserID:          "40",
            Comment:             "buffer fatkheev",
        }
        buffer.AddFact(fact)
    }

    // Останавливаем буфер и ждем отправки всех фактов.
    buffer.Stop()
    log.Println("Все факты отправлены!")
}
