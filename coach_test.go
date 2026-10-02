package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

func coachFixture(t *testing.T) (*Store, *http.ServeMux) {
	t.Helper()
	s := &Store{path: filepath.Join(t.TempDir(), "data.json")}
	s.data.Students = []DrivingStudent{{ID: 1, Name: "Élève", Status: "Actif", DrivingTarget: 20, Package: 1000}}
	s.data.Instructors = []DrivingInstructor{{ID: 1, Name: "Moniteur", Status: "Actif"}, {ID: 2, Name: "Ancien", Status: "Actif"}}
	s.data.Vehicles = []DrivingVehicle{{ID: 1, Status: "Disponible"}}
	s.data.DrivingConfig = DrivingConfig{OpenTime: "08:00", CloseTime: "18:00", DefaultLessonMinutes: 60}
	mux := http.NewServeMux()
	registerDrivingRoutes(mux, s, &AuthManager{})
	return s, mux
}

type coachRecorder struct {
	Code    int
	Body    bytes.Buffer
	headers http.Header
}

func (w *coachRecorder) Header() http.Header  { return w.headers }
func (w *coachRecorder) WriteHeader(code int) { w.Code = code }
func (w *coachRecorder) Write(b []byte) (int, error) {
	if w.Code == 0 {
		w.Code = 200
	}
	return w.Body.Write(b)
}
func coachPost(mux *http.ServeMux, url string, v any) *coachRecorder {
	b, _ := json.Marshal(v)
	r, _ := http.NewRequest("POST", "http://local.test"+url, bytes.NewReader(b))
	w := &coachRecorder{headers: http.Header{}}
	mux.ServeHTTP(w, r)
	return w
}
func TestCoachClosingAndSingleInstructor(t *testing.T) {
	s, m := coachFixture(t)
	lesson := DrivingLesson{StudentID: 1, InstructorID: 1, VehicleID: 1, Start: "2026-10-01T17:30", Duration: 60, Kind: "Conduite", Status: "Planifiée"}
	if w := coachPost(m, "/api/driving/lesson", lesson); w.Code != 409 {
		t.Fatalf("closing: %d %s", w.Code, w.Body.String())
	}
	lesson.Start = "2026-10-01T10:00"
	if w := coachPost(m, "/api/driving/lesson", lesson); w.Code != 200 {
		t.Fatal(w.Body)
	}
	s.data.Students = append(s.data.Students, DrivingStudent{ID: 2, Name: "Autre", Status: "Actif"})
	lesson.StudentID = 2
	lesson.InstructorID = 2
	if w := coachPost(m, "/api/driving/lesson", lesson); w.Code != 409 {
		t.Fatal("overlap accepted", w.Body)
	}
	lesson.Start = "2026-10-01T11:00"
	if w := coachPost(m, "/api/driving/lesson", lesson); w.Code != 200 {
		t.Fatal("adjacent lesson", w.Body)
	}
}
func TestCoachFollowupAtomicAndIdempotent(t *testing.T) {
	s, m := coachFixture(t)
	req := map[string]any{"exam": DrivingExam{StudentID: 1, Type: "Conduite", Date: "2026-09-12", Result: "Échec", RequestID: "test-request"}, "action": "rattrapage", "hours": 0, "price": 25}
	if w := coachPost(m, "/api/driving/exam-followup", req); w.Code != 400 {
		t.Fatal(w.Body)
	}
	if len(s.data.Exams) != 0 {
		t.Fatal("invalid follow-up saved an exam")
	}
	req["hours"] = 0.5
	for i := 0; i < 2; i++ {
		if w := coachPost(m, "/api/driving/exam-followup", req); w.Code != 200 {
			t.Fatal(w.Body)
		}
	}
	if len(s.data.Exams) != 1 || s.data.Students[0].DrivingTarget != 20.5 || s.data.Students[0].Package != 1025 {
		t.Fatal("duplicated or rounded follow-up")
	}
}
func TestCoachQueueAndFailureRollback(t *testing.T) {
	s, m := coachFixture(t)
	if w := coachPost(m, "/api/driving/queue", []int{1, 1}); w.Code != 400 {
		t.Fatal("duplicate queue accepted")
	}
	if w := coachPost(m, "/api/driving/queue", []int{1}); w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w := coachPost(m, "/api/driving/config", DrivingConfig{OpenTime: "08:00", CloseTime: "18:00", DefaultLessonMinutes: 60}); w.Code != 200 {
		t.Fatal(w.Body)
	}
	if len(s.data.DrivingConfig.QueueOrder) != 1 {
		t.Fatal("config erased queue")
	}
	s.path = filepath.Join(t.TempDir(), "missing", "data.json")
	req := map[string]any{"exam": DrivingExam{StudentID: 1, Type: "Conduite", Date: "2026-09-12", Result: "Échec"}, "action": "rattrapage", "hours": 2}
	if w := coachPost(m, "/api/driving/exam-followup", req); w.Code != 500 {
		t.Fatal(w.Body)
	}
	if len(s.data.Exams) != 0 || s.data.Students[0].DrivingTarget != 20 {
		t.Fatal("disk failure left mutations")
	}
}
