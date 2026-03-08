use std::collections::HashMap;

use crate::event::Event;
use crate::types::SubscriptionPattern;

struct Sub {
    _id: u64,
    pattern: SubscriptionPattern,
    handler: Box<dyn Fn(&Event)>,
}

pub struct EventBus {
    subs: HashMap<u64, Sub>,
    next_id: u64,
    closed: bool,
}

impl EventBus {
    pub fn new() -> Self {
        Self { subs: HashMap::new(), next_id: 0, closed: false }
    }

    pub fn subscribe(
        &mut self,
        pattern: SubscriptionPattern,
        handler: impl Fn(&Event) + 'static,
    ) -> u64 {
        if self.closed { return 0; }
        self.next_id += 1;
        let id = self.next_id;
        self.subs.insert(id, Sub { _id: id, pattern, handler: Box::new(handler) });
        id
    }

    pub fn unsubscribe(&mut self, sub_id: u64) {
        self.subs.remove(&sub_id);
    }

    pub fn publish(&self, event: &Event) {
        if self.closed { return; }
        for sub in self.subs.values() {
            if sub.pattern.matches(&event.event_type) {
                (sub.handler)(event);
            }
        }
    }

    pub fn close(&mut self) {
        self.closed = true;
        self.subs.clear();
    }
}

impl Default for EventBus {
    fn default() -> Self { Self::new() }
}
