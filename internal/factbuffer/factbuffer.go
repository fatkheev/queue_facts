package factbuffer

import (
    "io"
    "log"
    "net/http"
    "net/url"
    "strings"
    "time"
)

type Fact struct {
    PeriodStart         string
    PeriodEnd           string
    PeriodKey           string
    IndicatorToMoID     string
    IndicatorToMoFactID string
    Value               string
    FactTime            string
    IsPlan              string
    AuthUserID          string
    Comment             string
    EnqueuedAt          time.Time
}

// Структура очереди для фактов типа буфер
// В буфере факты сначала накапливаются, а потом по одному отправляются на сервер.
type FactBuffer struct {
    queue  chan Fact     // Канал для хранения фактов
    client *http.Client  // клиент для отправки на сервер
    done   chan struct{} // Канал для сигналов, что все факты обработаны
}

// Немного похардкодим
const apiURL = "https://development.kpi-drive.ru/_api/facts/save_fact"
const bearerToken = "48ab34464a5573519725deb5865cc74c"

// Cоздаеv очередь с нужной вместимостью, настраиваеv клиент с таймаутом 5 секунд и запускаеv воркера в отдельном потоке, которая будет брать факты из очереди и отправлять  на сервер.
func NewFactBuffer(bufferSize int) *FactBuffer {
    fb := &FactBuffer{
        queue:  make(chan Fact, bufferSize),
        client: &http.Client{Timeout: 5 * time.Second},
        done:   make(chan struct{}),
    }
    go fb.startWorker()
    return fb
}

// По очереди берем факты из очереди и отправляем на сервер.
func (fb *FactBuffer) startWorker() {
    // Пока в очереди есть факты, работаем с ними
    for fact := range fb.queue {
        // Если факт в очереди больше 5 минут, считаем его устаревшим и пропускаем
        if time.Since(fact.EnqueuedAt) > 5*time.Minute {
            log.Printf("Отбрасываем факт (находился в буфере больше 5 мин): %+v", fact)
            continue
        }
        values := url.Values{}
        values.Set("period_start", fact.PeriodStart)
        values.Set("period_end", fact.PeriodEnd)
        values.Set("period_key", fact.PeriodKey)
        values.Set("indicator_to_mo_id", fact.IndicatorToMoID)
        values.Set("indicator_to_mo_fact_id", fact.IndicatorToMoFactID)
        values.Set("value", fact.Value)
        values.Set("fact_time", fact.FactTime)
        values.Set("is_plan", fact.IsPlan)
        values.Set("auth_user_id", fact.AuthUserID)
        values.Set("comment", fact.Comment)

        // Создаем новый запрос для отправки
        req, err := http.NewRequest("POST", apiURL, strings.NewReader(values.Encode()))
        if err != nil {
            // Если не удалось, логируем ошибку и переходим к следующему факту
            log.Printf("Ошибка создания HTTP-запроса: %v", err)
            continue
        }
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        req.Header.Set("Authorization", "Bearer "+bearerToken)

        log.Printf("Отправка факта: %+v", fact)
        resp, err := fb.client.Do(req)
        if err != nil {
            log.Printf("Ошибка отправки факта (нет ответа от сервера): %v", err)
            continue
        }
        // Проверяем статус ответа
        if resp.StatusCode != http.StatusOK {
            // Если статус не 200, читаем и логируем ошибку
            _, _ = io.Copy(io.Discard, resp.Body)
            _ = resp.Body.Close()
            log.Printf("Сервер вернул код %d, пропускаем факт", resp.StatusCode)
            continue
        }
        // Если 200, факт отправлен
        _, _ = io.Copy(io.Discard, resp.Body) // Читаем ответ без использования
        _ = resp.Body.Close()                 // Закрываем тело ответа
        log.Printf("Факт успешно отправлен (статус 200 OK).")
    }
    // Если очередь закрыта и все факты обработаны, радуемся успеху
    log.Println("Буфер: больше нет фактов для отправки, работа воркера завершается.")
    close(fb.done)
}

// Когда  факт поступает, ставим на него  время и кладем в очередь. 
func (fb *FactBuffer) AddFact(fact Fact) {
    fact.EnqueuedAt = time.Now()
    fb.queue <- fact
    log.Printf("Факт добавлен в очередь буфера: %+v", fact)
}

// Останавливаем буфер, чтобы новые факты не добавлялись и ждем, пока воркер завершит обработку.
func (fb *FactBuffer) Stop() {
    close(fb.queue)
    <-fb.done
}
