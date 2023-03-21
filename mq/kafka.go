package mq

import (
	"github.com/opensourceways/kafka-lib/kafka"
	"github.com/opensourceways/kafka-lib/mq"
	"github.com/sirupsen/logrus"
)

var instance *serviceImpl

func Init(cfg *Config, log *logrus.Entry) error {
	err := kafka.Init(
		mq.Addresses(cfg.Address),
		mq.Log(log),
	)
	if err != nil {
		return err
	}

	if err = kafka.Connect(); err != nil {
		return err
	}

	instance = &serviceImpl{}

	return nil
}

func Exit() {
	if instance != nil {
		instance.unsubscribe()
	}

	if err := kafka.Disconnect(); err != nil {
		logrus.Errorf("exit kafka, err:%v", err)
	}
}

type Handler func([]byte) error

type serviceImpl struct {
	subscribers []mq.Subscriber
}

func Subscriber() *serviceImpl {
	return instance
}

func (impl *serviceImpl) Publish(topic string, msg []byte) error {
	return kafka.Publish(topic, &mq.Message{
		Body: msg,
	})
}

func (impl *serviceImpl) unsubscribe() {
	s := impl.subscribers
	for i := range s {
		if err := s[i].Unsubscribe(); err != nil {
			logrus.Errorf(
				"failed to unsubscribe to topic:%s, err:%v",
				s[i].Topic(), err,
			)
		}
	}
}

func (impl *serviceImpl) Subscribe(group string, handlers map[string]Handler) error {
	for topic, h := range handlers {
		s, err := impl.registerHandler(topic, group, h)
		if err != nil {
			return err
		}

		if s != nil {
			impl.subscribers = append(impl.subscribers, s)
		} else {
			logrus.Infof("does not subscribe topic:%s", topic)
		}
	}

	return nil
}

func (impl *serviceImpl) registerHandler(topic, group string, h Handler) (mq.Subscriber, error) {
	if h == nil {
		return nil, nil
	}

	return kafka.Subscribe(topic, group+topic, func(e mq.Event) error {
		msg := e.Message()
		if msg == nil {
			return nil
		}

		return h(msg.Body)
	})
}
