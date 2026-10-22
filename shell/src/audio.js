// The piezo buzzer.
//
// The P1 has a single piezoelectric element behind a hole in the case. It
// makes a harsh, thin sound with very little body to it, so this uses a
// square wave pushed through a band pass rather than anything musical.
//
// There is no mute here. The pet decides when to make a sound, and a muted
// pet keeps its mouth shut at the source, so nothing arrives to play.

(function () {
  let context = null;

  // audioContext returns the shared context, waking it up if the browser has
  // suspended it.
  function audioContext() {
    if (!context) {
      const AudioContext = window.AudioContext || window.webkitAudioContext;
      if (!AudioContext) {
        return null;
      }
      context = new AudioContext();
    }
    if (context.state === "suspended") {
      context.resume();
    }
    return context;
  }

  // beep sounds the piezo at a frequency for a number of milliseconds.
  function beep(frequency, duration) {
    const audio = audioContext();
    if (!audio) {
      return;
    }

    const now = audio.currentTime;
    const seconds = Math.max(0.02, (duration || 60) / 1000);

    const oscillator = audio.createOscillator();
    oscillator.type = "square";
    oscillator.frequency.setValueAtTime(frequency, now);

    // The piezo has almost no envelope of its own. It starts hard and stops
    // hard, which is what makes it sound so thin.
    const gain = audio.createGain();
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.14, now + 0.004);
    gain.gain.setValueAtTime(0.14, now + seconds);
    gain.gain.linearRampToValueAtTime(0, now + seconds);

    // The band pass takes the worst of the edges off.
    const filter = audio.createBiquadFilter();
    filter.type = "bandpass";
    filter.frequency.setValueAtTime(Math.min(frequency * 1.7, 12000), now);
    filter.Q.setValueAtTime(0.8, now);

    oscillator.connect(gain);
    gain.connect(filter);
    filter.connect(audio.destination);

    oscillator.start(now);
    oscillator.stop(now + seconds + 0.02);
  }

  // click is the sound of a button going down. It is not from the piezo, it
  // is the plastic.
  function click() {
    const audio = audioContext();
    if (!audio) {
      return;
    }

    const now = audio.currentTime;

    const oscillator = audio.createOscillator();
    oscillator.type = "triangle";
    oscillator.frequency.setValueAtTime(1900, now);
    oscillator.frequency.exponentialRampToValueAtTime(500, now + 0.02);

    const gain = audio.createGain();
    gain.gain.setValueAtTime(0.09, now);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.035);

    oscillator.connect(gain);
    gain.connect(audio.destination);

    oscillator.start(now);
    oscillator.stop(now + 0.05);
  }

  // unlock wakes the audio up. Browsers keep it asleep until the user has
  // touched the page.
  function unlock() {
    const audio = audioContext();
    if (audio && audio.state === "suspended") {
      audio.resume();
    }
  }

  window.TamaAudio = {
    beep,
    click,
    unlock,
  };
})();
