package errors

import "errors"

// ErrTemporary обозначает временную ошибку, которую можно повторить.
// Обработчик должен отправить сообщение обратно на повторную попытку (retry).
var ErrTemporary = errors.New("temporary error")

// ErrFatal обозначает фатальную ошибку, после которой повторять операцию бессмысленно.
// Обработчик должен отправить сообщение в финальную error-очередь.
var ErrFatal = errors.New("fatal error")
