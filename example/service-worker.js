"use strict";

self.addEventListener("install", function (event) {
  event.waitUntil(self.skipWaiting());
});

self.addEventListener("activate", function (event) {
  event.waitUntil(clients.claim());
});

self.addEventListener("push", function (event) {
  if (!(self.Notification && self.Notification.permission === "granted")) {
    return;
  }

  let message = {};
  try {
    message = event.data ? event.data.json() : {};
  } catch (_) {
    message = {
      title: "Nova notificação",
      options: { body: event.data ? event.data.text() : "" },
    };
  }

  const title = message.title || "Nova notificação";
  const options = message.options || {};

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener("notificationclick", function (event) {
  event.notification.close();

  const data = event.notification.data || {};
  const url = data.url || "/";

  event.waitUntil(
    clients
      .matchAll({ type: "window", includeUncontrolled: true })
      .then(function (clientList) {
        for (let i = 0; i < clientList.length; i += 1) {
          const client = clientList[i];
          if ("focus" in client) {
            return client.focus();
          }
        }

        if (clients.openWindow) {
          return clients.openWindow(url);
        }

        return undefined;
      }),
  );
});

self.addEventListener("notificationclose", function (event) {
  event.waitUntil(Promise.resolve());
});
