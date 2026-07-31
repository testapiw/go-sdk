type Handler struct {
    Policy Policy
}

func (h *Handler) Handle(e *transport.Event) transport.Decision